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

package arp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/tdrn-org/netscanner/internal/cache"
	"github.com/tdrn-org/netscanner/internal/cache/memory"
)

type Cache struct {
	cache cache.KeyValue[netip.Addr, net.HardwareAddr]
}

func NewCache(ttl time.Duration) (*Cache, error) {
	cache, err := memory.NewKeyValue(0, ttl, func(_ context.Context, _ netip.Addr) (net.HardwareAddr, error) { return nil, cache.ErrNotFound })
	if err != nil {
		return nil, fmt.Errorf("failed to create ARP cache (cause: %w)", err)
	}
	c := &Cache{
		cache: cache,
	}
	return c, nil
}

func (c *Cache) Get(ctx context.Context, address netip.Addr) net.HardwareAddr {
	hardwareAddress, _ := c.cache.Get(ctx, address)
	return hardwareAddress
}

func (c *Cache) Put(ctx context.Context, address netip.Addr, hardwareAddress net.HardwareAddr) {
	c.cache.Put(ctx, address, hardwareAddress)
}
