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

package model_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/sensor"
)

func TestProfile(t *testing.T) {
	datastore := newDatastore(t)
	defer datastore.Close()

	profile := newTestProfile(datastore)
	err := profile.Insert(t.Context())
	require.NoError(t, err)
}

func newTestProfile(datastore *database.Driver) *model.Profile {
	profile := model.NewProfile(datastore, "test_profile")
	logMatcher := model.NewLogMatcher("test_log_matcher", "test_tokenizer")
	profile.LogMatchers[logMatcher.Name] = logMatcher
	logMatcherEntry := model.NewLogMatcherEntry("sshd", sensor.EventTypeDenied, "Connection reset by authenticating user {User} {IP} port {Any} [preauth]")
	logMatcher.Entries = append(logMatcher.Entries, logMatcherEntry)
	logMatcherEntry = model.NewLogMatcherEntry("sshd", sensor.EventTypeGranted, "Accepted publickey for {User} from {IP} port {Any} ssh2: RSA")
	logMatcher.Entries = append(logMatcher.Entries, logMatcherEntry)
	syslogSensor := model.NewSyslogSensor("test_syslog_sensor", true, "tcp", "localhost:0", logMatcher.Name)
	profile.SyslogSensors[syslogSensor.Name] = syslogSensor
	return profile
}
