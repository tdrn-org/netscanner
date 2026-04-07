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

package probe_test

import (
	"fmt"
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/probe"
)

func runProbe[R probe.Result](t *testing.T, probeRunner probe.Runner[R], host string) {
	addrs, err := net.DefaultResolver.LookupHost(t.Context(), host)
	require.NoError(t, err)
	for _, addr := range addrs {
		address, err := netip.ParseAddr(addr)
		require.NoError(t, err)
		result := probeRunner.Run(t.Context(), address)
		require.NoError(t, result.Error())
		fmt.Println(result)
	}
}
