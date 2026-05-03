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

package network_test

import (
	"bytes"
	"net/netip"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/network"
)

func TestNamesSaveLoad(t *testing.T) {
	names1 := buildTestNames(t)
	buffer := &bytes.Buffer{}
	written, err := names1.Save(buffer)
	require.NoError(t, err)
	require.Equal(t, 70, written)

	names2 := network.NewNames()
	err = names2.Load(buffer)
	require.NoError(t, err)

	require.Equal(t, names1.Names(), names2.Names())
}

func TestNamesMatch(t *testing.T) {
	names := buildTestNames(t)
	// Intra
	require.Equal(t, "Intra", names.Match(netip.MustParseAddr("192.168.1.2")))
	require.Equal(t, "Intra", names.Match(netip.MustParseAddr("fd01::2")))
	// VPN
	require.Equal(t, "VPN", names.Match(netip.MustParseAddr("192.168.2.2")))
	require.Equal(t, "VPN", names.Match(netip.MustParseAddr("fd02::2")))
	// GlobalUnicast
	require.Equal(t, network.GlobalUnicast, names.Match(netip.MustParseAddr("1.1.1.1")))
	require.Equal(t, network.GlobalUnicast, names.Match(netip.MustParseAddr("2000::1")))
}

func TestNamesDefaults(t *testing.T) {
	names := network.NewNames()
	// Unspecified
	require.Equal(t, network.Unspecified, names.Match(netip.MustParseAddr("0.0.0.0")))
	require.Equal(t, network.Unspecified, names.Match(netip.IPv6Unspecified()))
	// Loopback
	require.Equal(t, network.Loopback, names.Match(netip.MustParseAddr("127.0.0.1")))
	require.Equal(t, network.Loopback, names.Match(netip.IPv6Loopback()))
	// LocalMulticast
	require.Equal(t, network.LocalMulticast, names.Match(netip.MustParseAddr("224.0.0.1")))
	require.Equal(t, network.LocalMulticast, names.Match(netip.IPv6LinkLocalAllNodes()))
	// Multicast
	require.Equal(t, network.Multicast, names.Match(netip.MustParseAddr("224.0.1.1")))
	require.Equal(t, network.Multicast, names.Match(netip.MustParseAddr("ff00::1")))
	// Private
	require.Equal(t, network.Private, names.Match(netip.MustParseAddr("192.168.1.1")))
	require.Equal(t, network.Private, names.Match(netip.MustParseAddr("fd00::1")))
	// LocalUnicast
	require.Equal(t, network.LocalUnicast, names.Match(netip.MustParseAddr("169.254.1.1")))
	require.Equal(t, network.LocalUnicast, names.Match(netip.MustParseAddr("fe80::1")))
	// GlobalUnicast
	require.Equal(t, network.GlobalUnicast, names.Match(netip.MustParseAddr("1.1.1.1")))
	require.Equal(t, network.GlobalUnicast, names.Match(netip.MustParseAddr("2000::1")))
}

func buildTestNames(t *testing.T) *network.Names {
	names := network.NewNames()
	file, err := os.Open("testdata/networks.txt")
	require.NoError(t, err)
	defer file.Close()
	err = names.Load(file)
	require.NoError(t, err)
	return names
}
