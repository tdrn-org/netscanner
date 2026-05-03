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

package cache

import (
	"context"
	"errors"
	"io"
)

type Name string

func (n Name) String() string {
	return string(n)
}

var ErrNotFound error = errors.New("not found")

type LoadFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

func NotFound[K comparable, V any](value V) LoadFunc[K, V] {
	return func(_ context.Context, _ K) (V, error) {
		return value, ErrNotFound
	}
}

type Cache[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, bool)
}

type NoCache[K comparable, V any] struct {
	Value V
	Found bool
}

func (c *NoCache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	return c.Value, c.Found
}

type KeyValue[K comparable, V any] interface {
	Cache[K, V]
	Put(ctx context.Context, key K, value V)
	io.Closer
}
