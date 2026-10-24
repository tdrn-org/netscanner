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
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing/fstest"
	"time"

	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-httpserver/csp"
	"github.com/tdrn-org/netscanner/internal/i18n"
	"golang.org/x/text/language"
)

//go:embed all:build/*
var buildFS embed.FS

//go:embed all:messages/*
var messagesFS embed.FS

// basePathPlaceholder is the literal placeholder baked into the SvelteKit build
// (see svelte.config.js). MountStatics replaces every occurrence with the
// runtime base path, so the same embedded bundle can be served under any prefix.
const basePathPlaceholder = "/__NETSCANNER_BASE_PATH__"

// defaultDocument is the entry document of the web UI. It is served for the
// document root and as fallback for every client side route.
const defaultDocument = "index.html"

// immutablePathPrefix is the path prefix SvelteKit uses for content hashed
// build artifacts. Documents below this prefix may be cached indefinitely.
const immutablePathPrefix = "/_app/immutable/"

// BasePath derives the URL prefix used to host the application from the public URL.
// Returns "" for root hosting, or a leading-slash path with no trailing slash (e.g. "/netscanner").
func BasePath(publicURL *url.URL) string {
	if publicURL == nil {
		return ""
	}
	trimmed := strings.Trim(publicURL.Path, "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed
}

func MountStatics(server *httpserver.Instance, basePath string) error {
	handler, err := newStaticHandler(server.BaseURL().Scheme, basePath)
	if err != nil {
		return err
	}
	if basePath != "" {
		handler = http.StripPrefix(basePath, handler)
	}
	server.Handle(basePath+"/", handler)
	return nil
}

// newStaticHandler builds the handler serving the embedded web document. The
// returned handler expects request paths relative to the document root (i.e.
// without any base path prefix).
func newStaticHandler(scheme string, basePath string) (http.Handler, error) {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		return nil, fmt.Errorf("unexpected web document structure (cause: %w)", err)
	}
	docs, err := rewriteBasePath(sub.(fs.ReadDirFS), basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite web document base path (cause: %w)", err)
	}
	contentSecurityPolicy := &csp.ContentSecurityPolicy{
		BaseUri:       []string{csp.SrcSelf},
		FormAction:    []string{csp.SrcSelf, scheme + ":"},
		FrameAncestor: []string{csp.SrcNone},
		DefaultSrc:    []string{csp.SrcNone},
		ConnectSrc:    []string{csp.SrcSelf},
		ScriptSrc:     []string{csp.SrcSelf},
		StyleSrc:      []string{csp.SrcSelf, csp.SrcUnsafeInline},
		ImgSrc:        []string{csp.SrcSelf, csp.SrcData},
		ObjectSrc:     []string{csp.SrcNone},
	}
	err = contentSecurityPolicy.AddHashes(csp.HashAlgSHA256, docs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate csp hashes (cause: %w)", err)
	}
	handler := &staticHandler{
		docs:       docs,
		fileServer: http.FileServerFS(docs),
	}
	return httpserver.HeaderHandler(handler, contentSecurityPolicy.Header(), cacheControl()), nil
}

// cacheControl classifies the requested documents into cacheable assets and
// documents that have to be revalidated.
//
// SvelteKit emits content-hashed file names below _app/immutable, so those can
// be cached indefinitely. Everything else - the entry document index.html, the
// build manifest _app/version.json and every document addressable by a stable
// name - must be revalidated, otherwise a new build is not picked up.
func cacheControl() httpserver.Header {
	return httpserver.HeaderFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, immutablePathPrefix) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
	})
}

// staticHandler serves the embedded web document and implements the SPA
// fallback for client-side routes.
type staticHandler struct {
	docs       fs.FS
	fileServer http.Handler
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestPath := strings.TrimPrefix(r.URL.Path, "/")
	path := requestPath
	if path == "" || strings.HasSuffix(path, "/") {
		path += defaultDocument
	}
	if _, err := fs.Stat(h.docs, path); err == nil {
		h.fileServer.ServeHTTP(w, r)
		return
	}
	if isDocumentRoute(requestPath) {
		h.serveDefaultDocument(w, r)
		return
	}
	h.serveNotFound(w)
}

// isDocumentRoute reports whether a request path which is not part of the web
// document is a client side route that has to be answered with the entry
// document. Missing assets are excluded: a file with an extension - typically a
// chunk of a previous build - must yield 404, because answering it with the
// entry document makes the browser parse HTML as script or stylesheet.
func isDocumentRoute(requestPath string) bool {
	return requestPath == "" || filepath.Ext(requestPath) == ""
}

// serveDefaultDocument writes the entry document. It is served directly
// instead of routing the request through http.FileServer: file servers answer
// requests for an index document with a redirect to the enclosing directory,
// which - relative to the original client side route - leads back into the
// fallback and finally results in a redirect loop or on the start page.
func (h *staticHandler) serveDefaultDocument(w http.ResponseWriter, r *http.Request) {
	content, err := fs.ReadFile(h.docs, defaultDocument)
	if err != nil {
		h.serveNotFound(w)
		return
	}
	http.ServeContent(w, r, defaultDocument, time.Time{}, bytes.NewReader(content))
}

// serveNotFound reports a document which is not part of the web document. The
// cache directive is reset: a failed request must not be cached, even when its
// path belongs to the immutable assets.
func (h *staticHandler) serveNotFound(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

// rewriteBasePath returns an in-memory FS where every occurrence of
// basePathPlaceholder is substituted with basePath. Binary files (without
// the placeholder) are copied through unchanged.
func rewriteBasePath(src fs.ReadDirFS, basePath string) (fs.ReadDirFS, error) {
	placeholder := []byte(basePathPlaceholder)
	replacement := []byte(basePath)
	out := fstest.MapFS{}
	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[path] = &fstest.MapFile{
			Data:    bytes.ReplaceAll(data, placeholder, replacement),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

var messageTables map[language.Tag]map[string]string

func Message(locale language.Tag, key string) string {
	messageTable, exists := messageTables[i18n.Match(locale)]
	if exists {
		message, exists := messageTable[key]
		if exists {
			return message
		}
	}
	defaultMessageTable, exists := messageTables[i18n.DefaultLocale()]
	if !exists {
		return ""
	}
	return defaultMessageTable[key]
}

func init() {
	initMessageTables()
}

func initMessageTables() {
	messagesDir := "messages"
	entries, err := messagesFS.ReadDir(messagesDir)
	if err != nil {
		panic(err)
	}
	messageTables = make(map[language.Tag]map[string]string)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		entryName := entry.Name()
		entryExt := filepath.Ext(entryName)
		if entryExt != ".json" {
			continue
		}
		locale := language.MustParse(strings.TrimSuffix(entryName, entryExt))
		messageTableData, err := messagesFS.ReadFile(filepath.Join(messagesDir, entryName))
		if err != nil {
			panic(err)
		}
		messageTable := make(map[string]string)
		err = json.Unmarshal(messageTableData, &messageTable)
		if err != nil {
			panic(err)
		}
		messageTables[locale] = messageTable
	}
}
