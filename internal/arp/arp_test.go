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

package arp_test

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/internal/arp"
)

func TestCache(t *testing.T) {
	cache, err := arp.NewCache(5 * time.Second)
	require.NoError(t, err)

	address := netip.IPv6Loopback()
	hardwareAddress, err := net.ParseMAC("01:02:03:04:05:06")
	require.NoError(t, err)

	// No hit
	found := cache.Get(t.Context(), address)
	require.Nil(t, found)

	// Hit
	cache.Put(t.Context(), address, hardwareAddress)
	found = cache.Get(t.Context(), address)
	require.Equal(t, hardwareAddress, found)

	// No hit (expired)
	time.Sleep(10 * time.Second)
	found = cache.Get(t.Context(), address)
	require.Nil(t, found)
}
