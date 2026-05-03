//go:build tools
// +build tools

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

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/tdrn-org/netscanner/ouidb"
)

func main() {
	switch os.Args[1] {
	case "ouidb":
		generateOuidb()
	}
}

func generateOuidb() {
	url := os.Args[2]
	rsp, err := http.Get(url)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to access %s (cause: %w)", url, err))
	}
	if rsp.StatusCode != http.StatusOK {
		log.Fatal(fmt.Errorf("failed to get %s (status: %s)", url, rsp.Status))
	}
	defer rsp.Body.Close()

	db := ouidb.NewPlainReader(rsp.Body)
	defer db.Close()

	indexWriter, err := ouidb.NewIndexWriter(".", "ouidb")
	if err != nil {
		log.Fatal(fmt.Errorf("failed create index (cause: %w)", err))
	}
	defer indexWriter.Close()

	for {
		vendor, err := db.ReadNext()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(fmt.Errorf("failed to read vendor (cause: %w)", err))
		}
		err = indexWriter.WriteVendor(vendor)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to write vendor (cause: %w)", err))
		}
	}
	err = indexWriter.WriteIndex()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to write index (cause: %w)", err))
	}
}
