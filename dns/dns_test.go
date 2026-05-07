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
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/dns"
	"github.com/tdrn-org/netscanner/dns/custom"
	"github.com/tdrn-org/netscanner/dns/system"
)

func TestSystemProvider(t *testing.T) {
	provider, err := dns.Open(&system.Config{})
	require.NoError(t, err)
	runProviderTests(t, provider)
}

func TestCustomProvider(t *testing.T) {
	provider, err := dns.Open(&custom.Config{
		Network: "udp",
		Address: "8.8.8.8:53",
	})
	require.NoError(t, err)
	runProviderTests(t, provider)
}

func runProviderTests(t *testing.T, provider dns.Provider) {
	// 8.8.8.8
	dns, err := provider.Lookup(t.Context(), netip.MustParseAddr("8.8.8.8"))
	require.NoError(t, err)
	require.NotEmpty(t, dns)

	// 2001:4860:4860::8888
	dns, err = provider.Lookup(t.Context(), netip.MustParseAddr("2001:4860:4860::8888"))
	require.NoError(t, err)
	require.NotEmpty(t, dns)

	// 0.0.0.0
	dns, err = provider.Lookup(t.Context(), netip.MustParseAddr("0.0.0.0"))
	require.NoError(t, err)
	require.Empty(t, dns)

}
