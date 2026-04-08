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

package netscanner_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner"
)

func TestNetscannerStartStop(t *testing.T) {
	config, err := netscanner.LoadConfig("testdata/netscanner.toml", true)
	require.NoError(t, err)
	server, err := netscanner.StartServer(t.Context(), config)
	require.NoError(t, err)
	go func() {
		err := server.Run(t.Context())
		if !errors.Is(err, http.ErrServerClosed) {
			require.NoError(t, err)
		}
	}()
	err = server.Ping(t.Context())
	require.NoError(t, err)
	err = server.Stop(t.Context())
	require.NoError(t, err)
}
