/*
 * Copyright 2025-2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package web

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-httpserver"
)

// testDocument mirrors the layout of a SvelteKit build: an entry document,
// content hashed artifacts below _app/immutable and a build manifest.
func testDocument() fs.ReadDirFS {
	return fstest.MapFS{
		"index.html":                           &fstest.MapFile{Data: []byte("<html>netscanner</html>")},
		"_app/version.json":                    &fstest.MapFile{Data: []byte(`{"version":"1"}`)},
		"_app/immutable/nodes/0.CLUgpc7M.js":   &fstest.MapFile{Data: []byte("console.log(0)")},
		"_app/immutable/assets/0.fGAPN8h8.css": &fstest.MapFile{Data: []byte("body{}")},
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	docs, err := rewriteBasePath(testDocument(), "/netscanner")
	require.NoError(t, err)
	handler := &staticHandler{docs: docs, fileServer: http.FileServerFS(docs)}
	return httpserver.HeaderHandler(handler, cacheControl())
}

// TestNewStaticHandler covers the handler built from the embedded web document:
// base path rewriting, content security policy and cache classification.
func TestNewStaticHandler(t *testing.T) {
	handler, err := newStaticHandler("https", "/netscanner")
	require.NoError(t, err)
	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{name: "document root", path: "/", expectedStatus: http.StatusOK},
		{name: "client side route", path: "/connections", expectedStatus: http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
			assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src")
			assert.Contains(t, rec.Body.String(), "/netscanner/")
		})
	}
}

// TestStaticHandlerWithoutDefaultDocument covers the case of a web document
// which does not contain an entry document at all.
func TestStaticHandlerWithoutDefaultDocument(t *testing.T) {
	docs, err := rewriteBasePath(fstest.MapFS{
		"_app/immutable/nodes/0.CLUgpc7M.js": &fstest.MapFile{Data: []byte("console.log(0)")},
	}, "")
	require.NoError(t, err)
	handler := httpserver.HeaderHandler(&staticHandler{docs: docs, fileServer: http.FileServerFS(docs)}, cacheControl())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/connections", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

// TestMountStatics covers mounting the web document below a base path.
func TestMountStatics(t *testing.T) {
	ctx := context.Background()
	server, err := httpserver.Listen(ctx, "tcp", "localhost:0")
	require.NoError(t, err)
	defer server.Close()
	require.NoError(t, MountStatics(server, "/netscanner"))
	require.NoError(t, MountStatics(server, ""))
}

func TestStaticHandler(t *testing.T) {
	handler := newTestHandler(t)
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
		expectedCache  string
	}{
		{
			name:           "document root",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
			expectedCache:  "no-cache",
		},
		{
			name:           "client side route",
			path:           "/connections",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
			expectedCache:  "no-cache",
		},
		{
			name:           "client side route with trailing slash",
			path:           "/connections/17/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html; charset=utf-8",
			expectedCache:  "no-cache",
		},
		{
			name:           "content hashed chunk",
			path:           "/_app/immutable/nodes/0.CLUgpc7M.js",
			expectedStatus: http.StatusOK,
			expectedCache:  "public, max-age=31536000, immutable",
		},
		{
			name:           "content hashed asset",
			path:           "/_app/immutable/assets/0.fGAPN8h8.css",
			expectedStatus: http.StatusOK,
			expectedCache:  "public, max-age=31536000, immutable",
		},
		{
			name:           "build manifest",
			path:           "/_app/version.json",
			expectedStatus: http.StatusOK,
			expectedCache:  "no-cache",
		},
		{
			name:           "missing chunk of a previous build",
			path:           "/_app/immutable/nodes/1.B8PKd4Dy.js",
			expectedStatus: http.StatusNotFound,
			expectedCache:  "no-store",
		},
		{
			name:           "missing document",
			path:           "/missing.png",
			expectedStatus: http.StatusNotFound,
			expectedCache:  "no-store",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, tc.expectedStatus, rec.Code)
			if tc.expectedType != "" {
				assert.Equal(t, tc.expectedType, rec.Header().Get("Content-Type"))
			}
			if tc.expectedCache != "" {
				assert.Equal(t, tc.expectedCache, rec.Header().Get("Cache-Control"))
			}
			if tc.expectedStatus == http.StatusOK && strings.HasSuffix(tc.path, "index.html") {
				assert.Contains(t, rec.Body.String(), "<html>")
			}
		})
	}
}

// TestStaticHandlerFallbackIsNotRedirecting guards the SPA fallback: the entry
// document has to be written directly. Routing the request through
// http.FileServer makes the server redirect to the enclosing directory, which
// relative to the original client side route leads back to the start page.
func TestStaticHandlerFallbackIsNotRedirecting(t *testing.T) {
	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/connections", nil))
	assert.Equal(t, http.StatusOK, rec.Code, "fallback must not redirect")
	assert.Empty(t, rec.Header().Get("Location"))
	assert.Contains(t, rec.Body.String(), "<html>netscanner</html>")
}

func TestStaticHandlerHead(t *testing.T) {
	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/connections", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}
