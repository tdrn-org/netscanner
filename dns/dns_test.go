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

package dns_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/dns"
)

func TestResolverProvider(t *testing.T) {
	provider := dns.NewResolverProvider(net.DefaultResolver, func(addr *netip.Addr) *netip.Addr { return addr })

	// 8.8.8.8
	addr := netip.MustParseAddr("8.8.8.8")
	info, err := provider.Lookup(t.Context(), &addr)
	require.NoError(t, err)
	require.NotEmpty(t, info.Name)

	// 0.0.0.0
	addr = netip.MustParseAddr("0.0.0.0")
	info, err = provider.Lookup(t.Context(), &addr)
	require.NoError(t, err)
	require.NotEmpty(t, info.Name)

	// 2001:4860:4860::8888
	addr = netip.MustParseAddr("2001:4860:4860::8888")
	info, err = provider.Lookup(t.Context(), &addr)
	require.NoError(t, err)
	require.NotEmpty(t, info.Name)
}
