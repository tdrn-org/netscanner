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

package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/internal/file"
)

func TestRegexpDecoder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	scanner := file.NewScanner(path, &file.RegexpDecoder{Pattern: regexp.MustCompile(`^(\d+): (.+)\n`)}, false)
	lines := make([]string, 10)
	for n := range lines {
		lines[n] = fmt.Sprintf("%d: %s\n", n+1, path)
	}
	go func() {
		file, err := os.Create(path)
		require.NoError(t, err)
		defer file.Close()
		for n := 0; n < len(lines); n = n + 2 {
			time.Sleep(time.Second)
			_, err := file.Write([]byte(lines[n]))
			require.NoError(t, err)
			_, err = file.Write([]byte(lines[n+1]))
			require.NoError(t, err)
		}
	}()
	for n := 0; n < len(lines); {
		_, match, err := scanner.Read()
		require.NoError(t, err)
		if match != nil {
			matchLine := fmt.Sprintf("%s: %s\n", match[1], match[2])
			require.Equal(t, lines[n], matchLine)
			n++
		}
	}
	time.Sleep(time.Second)
	_, line, err := scanner.Read()
	require.NoError(t, err)
	require.Empty(t, line)
	err = scanner.Close()
	require.NoError(t, err)
}
