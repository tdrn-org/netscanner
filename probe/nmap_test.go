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
	"encoding/xml"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/probe"
)

func TestNMapOutput(t *testing.T) {
	file, err := os.Open("testdata/testrun.xml")
	require.NoError(t, err)
	defer file.Close()
	decoder := xml.NewDecoder(file)
	run := &probe.NMapRun{}
	err = decoder.Decode(run)
	require.NoError(t, err)
	require.Len(t, run.Hosts, 1)
	host := run.Hosts[0]
	require.Len(t, host.Hostnames.Elements, 2)
	require.Len(t, host.Ports.Elements, 2)
	require.Len(t, host.OS.OSClass, 1)
	require.Len(t, host.OS.OSMatch, 1)
}

func TestNMapCmd(t *testing.T) {
	_, err := exec.LookPath(probe.DefaultNMapCommand)
	if err != nil {
		t.Skip("nmap not available")
	}
	nmap := probe.NewNMap()
	runProbe(t, nmap, "localhost")
}
