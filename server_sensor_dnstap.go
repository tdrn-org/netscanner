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
	"context"
	"fmt"
	"log/slog"

	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/dnstap"
)

type DnstapSensorConfig struct {
	Name   string       `toml:"name"`
	Source DnstapSource `toml:"source"`
	Path   string       `toml:"path"`
}

func (c *DnstapSensorConfig) String() string {
	return fmt.Sprintf(sensorStringFormatPath, "dnstap", c.Name, c.Path)
}

func (s *Server) addDnstapSensor(ctx context.Context, config *DnstapSensorConfig) (*sensor.Sensor, error) {
	s.logger.Info("adding sensor", slog.Any("sensor", config))
	var source *dnstap.Sensor
	var err error
	switch config.Source {
	case DnstapSourceFile:
		source, err = dnstap.PollFile(config.Path)
	case DnstapSourceSocket:
		source, err = dnstap.ListenSocket(config.Path)
	default:
		err = fmt.Errorf("unrecognized dnstap source '%s'", config.Source)
	}
	if err != nil {
		return nil, err
	}
	sensor := sensor.New(config.Name, source)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sensors[sensor.Name()] = sensor
	return sensor, nil
}

type DnstapSource string

const (
	DnstapSourceFile   DnstapSource = "file"
	DnstapSourceSocket DnstapSource = "socket"
)

var knownDnstapSources map[string]DnstapSource = map[string]DnstapSource{
	string(DnstapSourceFile):   DnstapSourceFile,
	string(DnstapSourceSocket): DnstapSourceSocket,
}

func (s *DnstapSource) Value() string {
	for value, dnstapSource := range knownDnstapSources {
		if *s == dnstapSource {
			return value
		}
	}
	slog.Warn("unexpected dnstap source", slog.Any("s", *s))
	return ""
}

func (s *DnstapSource) MarshalTOML() ([]byte, error) {
	return []byte(`"` + s.Value() + `"`), nil
}

func (s *DnstapSource) UnmarshalTOML(value any) error {
	dnstapSourceString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected dnstap source type %v", value)
	}
	dnstapSource, ok := knownDnstapSources[dnstapSourceString]
	if !ok {
		return fmt.Errorf("unknown dnstap source: '%s'", dnstapSourceString)
	}
	*s = dnstapSource
	return nil
}
