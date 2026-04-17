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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/sensor"
)

type Profile struct {
	Name          string                   `json:"name"`
	LogMatchers   map[string]*LogMatcher   `json:"log_matchers"`
	SyslogSensors map[string]*SyslogSensor `json:"syslog_sensors"`
}

func NewProfile(name string) *Profile {
	return &Profile{
		Name:          name,
		LogMatchers:   make(map[string]*LogMatcher, 0),
		SyslogSensors: make(map[string]*SyslogSensor, 0),
	}
}

func LoadProfile(r io.Reader) (*Profile, error) {
	profile := &Profile{}
	err := json.NewDecoder(r).Decode(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode profile (cause: %w)", err)
	}
	return profile, nil
}

func LoadProfileFile(file string) (*Profile, error) {
	profileReader, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed open profile file '%s' (cause: %w)", file, err)
	}
	defer profileReader.Close()
	return LoadProfile(profileReader)
}

func (p *Profile) Save(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(p)
	if err != nil {
		return fmt.Errorf("failed to encode profile '%s' (cause: %w)", p.Name, err)
	}
	return nil
}

func (p *Profile) SaveFile(file string) error {
	profileWriter, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed create/truncate profile file '%s' (cause: %w)", file, err)
	}
	defer profileWriter.Close()
	return p.Save(profileWriter)
}

func (p *Profile) toModel(datastore *database.Driver) *model.Profile {
	model := model.NewProfile(datastore, p.Name)
	for logMatcherName, logMatcher := range p.LogMatchers {
		model.LogMatchers[logMatcherName] = logMatcher.toModel(logMatcherName)
	}
	for syslogSensorName, syslogSensor := range p.SyslogSensors {
		model.SyslogSensors[syslogSensorName] = syslogSensor.toModel(syslogSensorName)
	}
	return model
}

type LogMatcher struct {
	Tokenizer string             `json:"tokenizer"`
	Entries   []*LogMatcherEntry `json:"entries"`
}

func (lm *LogMatcher) toModel(name string) *model.LogMatcher {
	model := model.NewLogMatcher(name, lm.Tokenizer)
	for _, lme := range lm.Entries {
		model.Entries = append(model.Entries, lme.toModel())
	}
	return model
}

type LogMatcherEntry struct {
	Service   string           `json:"service"`
	EventType sensor.EventType `json:"event_type"`
	Match     string           `json:"match"`
}

func (lme *LogMatcherEntry) toModel() *model.LogMatcherEntry {
	return model.NewLogMatcherEntry(lme.Service, lme.EventType, lme.Match)
}

type SyslogSensor struct {
	Enabled    bool   `json:"enabled"`
	Network    string `json:"network"`
	Address    string `json:"address"`
	LogMatcher string `json:"log_matcher"`
}

func (s *SyslogSensor) toModel(name string) *model.SyslogSensor {
	return model.NewSyslogSensor(name, s.Enabled, s.Network, s.Address, s.LogMatcher)
}
