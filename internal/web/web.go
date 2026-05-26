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
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		return fmt.Errorf("unexpected web document structure (cause: %w)", err)
	}
	docs, err := rewriteBasePath(sub.(fs.ReadDirFS), basePath)
	if err != nil {
		return fmt.Errorf("failed to rewrite web document base path (cause: %w)", err)
	}
	fileServer := http.FileServerFS(docs)
	// SPA fallback: serve index.html for unmatched paths (client-side routing)
	docsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try serving the file first
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := docs.Open(path)
		if err != nil {
			// File not found — fall back to index.html for SPA routing
			r.URL.Path = "/index.html"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
	contentSecurityPolicy := &csp.ContentSecurityPolicy{
		BaseUri:       []string{csp.SrcSelf},
		FormAction:    []string{csp.SrcSelf, server.BaseURL().Scheme + ":"},
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
		return fmt.Errorf("failed to generate csp hashes (cause: %w)", err)
	}
	cacheControl := httpserver.StaticHeader("Cache-Control", "public, max-age=86400, immutable")

	var handler http.Handler = docsHandler
	if basePath != "" {
		handler = http.StripPrefix(basePath, docsHandler)
	}

	server.Handle(basePath+"/", httpserver.HeaderHandler(handler, contentSecurityPolicy.Header(), cacheControl))
	return nil
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
