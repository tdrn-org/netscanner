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

package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/tdrn-org/netscanner/internal/cache"
)

const Name cache.Name = "memory"

type memoryKeyValue[K comparable, V any] struct {
	cache *otter.Cache[K, V]
	load  cache.LoadFunc[K, V]
}

func NewKeyValue[K comparable, V any](size int, ttl time.Duration, load cache.LoadFunc[K, V]) (cache.KeyValue[K, V], error) {
	options := &otter.Options[K, V]{
		MaximumSize:      size,
		ExpiryCalculator: otter.ExpiryAccessing[K, V](ttl),
	}
	cache, err := otter.New(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory cache (cause: %w)", err)
	}
	return &memoryKeyValue[K, V]{cache: cache, load: load}, nil
}

func (kv *memoryKeyValue[K, V]) Get(ctx context.Context, key K) (V, bool) {
	value, err := kv.cache.Get(ctx, key, otter.LoaderFunc[K, V](kv.load))
	return value, err == nil
}

func (kv *memoryKeyValue[K, V]) Put(ctx context.Context, key K, value V) {
	kv.cache.Set(key, value)
}

func (kv *memoryKeyValue[K, V]) Close() error {
	return nil
}
