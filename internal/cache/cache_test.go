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

package cache_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/internal/cache"
	"github.com/tdrn-org/netscanner/internal/cache/memory"
)

func TestMemoryKeyValue(t *testing.T) {
	kv, err := memory.NewKeyValue(0, time.Second, cache.NotFound[string](""))
	require.NoError(t, err)

	runKeyValueTest(t, kv)

	err = kv.Close()
	require.NoError(t, err)
}

func runKeyValueTest(t *testing.T, kv cache.KeyValue[string, string]) {
	const count = 1000
	for keyValue := range count {
		key := fmt.Sprintf("%d", keyValue)
		kv.Put(t.Context(), key, key)
		value, hit := kv.Get(t.Context(), key)
		require.True(t, hit)
		require.Equal(t, key, value)
	}
}
