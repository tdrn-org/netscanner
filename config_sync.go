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

package netscanner

import (
	"fmt"
	"log/slog"

	"github.com/tdrn-org/netscanner/mtls"
)

type SyncConfig struct {
	Mode    SyncMode `toml:"mode"`
	Address string   `toml:"address"`
	CRTFile string   `toml:"crt_file"`
	KeyFile string   `toml:"key_file"`
	CAFile  string   `toml:"ca_file"`
}

func (c *SyncConfig) loadCredentials() (*mtls.Credentials, error) {
	return mtls.LoadCredentials(c.CRTFile, c.KeyFile, c.CAFile)
}

type SyncMode string

const (
	SyncModeDisabled SyncMode = "disabled"
	SyncModeForward  SyncMode = "forward"
	SyncModeReceive  SyncMode = "receive"
)

var knownSyncModes map[string]SyncMode = map[string]SyncMode{
	string(SyncModeDisabled): SyncModeDisabled,
	string(SyncModeForward):  SyncModeForward,
	string(SyncModeReceive):  SyncModeReceive,
}

func (m *SyncMode) Value() string {
	for value, syncMode := range knownSyncModes {
		if *m == syncMode {
			return value
		}
	}
	slog.Warn("unexpected sync mode", slog.Any("m", *m))
	return ""
}

func (m *SyncMode) MarshalTOML() ([]byte, error) {
	return []byte(`"` + m.Value() + `"`), nil
}

func (m *SyncMode) UnmarshalTOML(value any) error {
	syncModeString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected sync mode type %v", value)
	}
	syncMode, ok := knownSyncModes[syncModeString]
	if !ok {
		return fmt.Errorf("unknown sync mode: '%s'", syncModeString)
	}
	*m = syncMode
	return nil
}
