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
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/logmatcher"
)

func TestAddressToken(t *testing.T) {
	validAddressToken := logmatcher.Token{Symbol: "::1"}
	require.Equal(t, logmatcher.TokenTypeAddress, validAddressToken.Type())
	require.NotNil(t, validAddressToken.AddressValue())
	invalidAddressToken := logmatcher.Token{Symbol: "x::1"}
	require.Equal(t, logmatcher.TokenTypeSymbol, invalidAddressToken.Type())
	require.Equal(t, netip.Addr{}, invalidAddressToken.AddressValue())
}

func TestHardwareAddressToken(t *testing.T) {
	validHardwareAddressToken := logmatcher.Token{Symbol: "00:00:5e:00:53:01"}
	require.Equal(t, logmatcher.TokenTypeHardwareAddress, validHardwareAddressToken.Type())
	require.NotNil(t, validHardwareAddressToken.HardwareAddressValue())
	invalidHardwareAddressToken := logmatcher.Token{Symbol: "xx:00:5e:00:53:01"}
	require.Equal(t, logmatcher.TokenTypeSymbol, invalidHardwareAddressToken.Type())
	require.Nil(t, invalidHardwareAddressToken.HardwareAddressValue())
}

func TestDefaultTokenizer(t *testing.T) {
	tokens := logmatcher.FieldsTokenizer("Connection reset by authenticating user root 127.0.0.0 port 63906 [preauth]")
	require.Len(t, tokens, 10)
	require.Equal(t, logmatcher.TokenTypeSymbol, tokens[0].Type())
	require.Equal(t, "Connection", tokens[0].Symbol)
	require.Equal(t, logmatcher.TokenTypeAddress, tokens[6].Type())
	require.Equal(t, "127.0.0.0", tokens[6].Symbol)
	require.Equal(t, netip.MustParseAddr("127.0.0.0"), tokens[6].AddressValue())
}
