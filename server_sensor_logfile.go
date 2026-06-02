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

	"github.com/tdrn-org/netscanner/config"
	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/logfile"
)

type LogfileSensorConfig struct {
	Name            string `toml:"name"`
	Path            string `toml:"path"`
	LogMatcherIndex string `toml:"log_matcher_index"`
	Regexp          struct {
		Pattern         config.RegexpSpec `toml:"pattern"`
		TimestampField  int               `toml:"timestamp_field"`
		TimestampLayout string            `toml:"timestamp_layout"`
		HostField       int               `toml:"host_field"`
		MessageField    int               `toml:"message_field"`
	} `toml:"regexp"`
}

func (c *LogfileSensorConfig) String() string {
	return fmt.Sprintf(sensorStringFormatPath, logfile.Name, c.Name, c.Path)
}

func (c *LogfileSensorConfig) regexpScanOptions() *logfile.RegexpScanOptions {
	return &logfile.RegexpScanOptions{
		Pattern:         c.Regexp.Pattern.Regexp,
		TimestampField:  c.Regexp.TimestampField,
		TimestampLayout: c.Regexp.TimestampLayout,
		HostField:       c.Regexp.HostField,
		MessageField:    c.Regexp.MessageField,
		Tail:            true, // for now we default to true (accept misses; before duplicates)
	}
}

func (s *Server) addLogfileSensor(ctx context.Context, config *LogfileSensorConfig) (*sensor.Sensor, error) {
	s.logger.Info("adding sensor", slog.Any("sensor", config))
	index, err := s.resolveLogMatcherIndex(ctx, config.LogMatcherIndex)
	if err != nil {
		return nil, err
	}
	source, err := logfile.ScanRegexp(index, config.Path, config.regexpScanOptions())
	if err != nil {
		return nil, err
	}
	sensor := sensor.New(config.Name, source)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sensors[sensor.Name()] = sensor
	return sensor, nil
}
