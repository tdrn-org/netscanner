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
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-httpserver/csp"
	"github.com/tdrn-org/netscanner/internal/i18n"
	"golang.org/x/text/language"
)

//go:embed all:build/*
var buildFS embed.FS

//go:embed all:messages/*
var messagesFS embed.FS

func MountStatics(server *httpserver.Instance, publicURL *url.URL) error {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		return fmt.Errorf("unexpected web document structure (cause: %w)", err)
	}
	docs := sub.(fs.ReadDirFS)
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

	// Derive base path from public URL
	basePath := ""
	if publicURL != nil {
		basePath = publicURL.Path
	}
	// Normalize base path: ensure leading slash, no trailing slash
	if basePath == "" {
		basePath = "/"
	} else {
		basePath = "/" + strings.Trim(basePath, "/")
	}

	var handler http.Handler = docsHandler
	if basePath != "/" {
		handler = http.StripPrefix(basePath, docsHandler)
	}

	server.Handle(basePath+"/", httpserver.HeaderHandler(handler, contentSecurityPolicy.Header(), cacheControl))
	return nil
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
