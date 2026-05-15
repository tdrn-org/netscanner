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
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/internal/file"
)

func TestJSONTypes(t *testing.T) {
	object := newTestJSON(t)

	// string level 1
	level, err := file.JSONValue[string](object, "level")
	require.NoError(t, err)
	require.Equal(t, "info", level)

	// float64 level 1
	ts, err := file.JSONValue[float64](object, "ts")
	require.NoError(t, err)
	require.Equal(t, 1778371934.4997208, ts)

	// bool level 3
	resumed, err := file.JSONValue[bool](object, "request", "tls", "resumed")
	require.NoError(t, err)
	require.True(t, resumed)
}

func TestJSONValue(t *testing.T) {
	object := newTestJSON(t)

	// string as string
	level, err := file.JSONValueToString(object, "level")
	require.NoError(t, err)
	require.Equal(t, "info", level)

	// time as number
	ts, err := file.JSONValueToTime("", object, "ts")
	require.NoError(t, err)
	require.Equal(t, time.Unix(1778371934, 499720812), ts)

	// int as number
	status, err := file.JSONValueToInt(object, "status")
	require.NoError(t, err)
	require.Equal(t, 200, status)
}

func TestJsonDecoder(t *testing.T) {
	scanner := file.NewScanner("testdata/log.json", &file.JSONDecoder{}, false)
	decodedCounter := 0
	for {
		_, decoded, err := scanner.Read()
		require.NoError(t, err)
		if decoded == nil {
			break
		}
		decodedCounter++
	}
	require.Equal(t, 4, decodedCounter)
}

func newTestJSON(t *testing.T) file.JSON {
	jsonFile, err := os.Open("testdata/log.json")
	require.NoError(t, err)
	var object file.JSON
	err = json.NewDecoder(jsonFile).Decode(&object)
	require.NoError(t, err)
	return object
}
