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

package logmatcher_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
)

func TestIndexSaveLoad(t *testing.T) {
	index1 := buildTestIndex(t)
	buffer := &bytes.Buffer{}
	written, err := index1.Save(buffer)
	require.NoError(t, err)
	require.Equal(t, 159, written)

	index2 := emptyTestIndex(t)
	err = index2.Load(buffer)
	require.NoError(t, err)

	require.Equal(t, index1.Size(), index2.Size())
}

func TestIndexResolveValues(t *testing.T) {
	tokenizer := logmatcher.FieldsTokenizer
	index := buildTestIndex(t)

	// Resolve empty message
	tokens1 := []logmatcher.Token{}
	resolved1 := index.ResolveValues(tokens1)
	require.Nil(t, resolved1)

	// Resolve matching message1
	tokens2 := tokenizer.Tokens("Connection reset by authenticating user admin 127.0.0.1 port 63906 [preauth]")
	resolved2 := index.ResolveValues(tokens2)
	require.NotNil(t, resolved2)
	require.Equal(t, sensor.EventTypeDenied, resolved2.EventType)
	require.Equal(t, "127.0.0.1", resolved2.Address.String())
	require.Nil(t, resolved2.HardwareAddress)
	require.Equal(t, "admin", resolved2.User)
	require.Equal(t, "sshd", resolved2.Service)

	// Resolve matching message1
	tokens3 := tokenizer.Tokens("Accepted publickey for root from ::1 port 41074 ssh2: RSA SHA256:xyz")
	resolved3 := index.ResolveValues(tokens3)
	require.NotNil(t, resolved3)
	require.Equal(t, sensor.EventTypeGranted, resolved3.EventType)
	require.Equal(t, "::1", resolved3.Address.String())
	require.Nil(t, resolved3.HardwareAddress)
	require.Equal(t, "root", resolved3.User)
	require.Equal(t, "sshd", resolved3.Service)

	// Resolve not matching message
	tokens4 := tokenizer.Tokens("Connection closed by authenticating user admin ::1 port 45054 [preauth]")
	resolved4 := index.ResolveValues(tokens4)
	require.Nil(t, resolved4)
}

func emptyTestIndex(_ *testing.T) *logmatcher.Index {
	return logmatcher.NewIndex("test")
}

func buildTestIndex(t *testing.T) *logmatcher.Index {
	index := emptyTestIndex(t)
	file, err := os.Open("testdata/index.txt")
	require.NoError(t, err)
	defer file.Close()
	err = index.Load(file)
	require.NoError(t, err)
	return index
}
