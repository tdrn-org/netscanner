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
	"github.com/tdrn-org/netscanner/sensor/accesslog"
	"github.com/tdrn-org/netscanner/sensor/logfile"
)

type AccesslogSensorConfig struct {
	Name       string      `toml:"name"`
	Path       string      `toml:"path"`
	AuthURIs   []string    `toml:"auth_uris"`
	IgnoreURIs []string    `toml:"ignore_uris"`
	Encoding   LogEncoding `toml:"encoding"`
	Regexp     struct {
		Pattern         config.RegexpSpec `toml:"pattern"`
		TimestampField  int               `toml:"timestamp_field"`
		TimestampLayout string            `toml:"timestamp_layout"`
		StatusField     int               `toml:"status_field"`
		AddressField    int               `toml:"address_field"`
		UserField       int               `toml:"user_field"`
		URIField        int               `toml:"uri_field"`
	} `toml:"regexp"`
	JSON struct {
		TimestampField  []string `toml:"timestamp_field"`
		TimestampLayout string   `toml:"timestamp_layout"`
		StatusField     []string `toml:"status_field"`
		AddressField    []string `toml:"address_field"`
		UserField       []string `toml:"user_field"`
		URIField        []string `toml:"uri_field"`
	} `toml:"json"`
}

func (c *AccesslogSensorConfig) String() string {
	return fmt.Sprintf(sensorStringFormatPath, logfile.Name, c.Name, c.Path)
}

func (c *AccesslogSensorConfig) regexpScanOptions() *accesslog.RegexpScanOptions {
	return &accesslog.RegexpScanOptions{
		ScanOptions: accesslog.ScanOptions{
			AuthURIs:   c.AuthURIs,
			IgnoreURIs: c.IgnoreURIs,
			Tail:       true, // for now we default to true (accept misses; before duplicates)
		},
		Pattern:         c.Regexp.Pattern.Regexp,
		TimestampField:  c.Regexp.TimestampField,
		TimestampLayout: c.Regexp.TimestampLayout,
		StatusField:     c.Regexp.StatusField,
		AddressField:    c.Regexp.AddressField,
		UserField:       c.Regexp.UserField,
	}
}

func (c *AccesslogSensorConfig) jsonScanOptions() *accesslog.JSONScanOptions {
	return &accesslog.JSONScanOptions{
		ScanOptions: accesslog.ScanOptions{
			AuthURIs:   c.AuthURIs,
			IgnoreURIs: c.IgnoreURIs,
			Tail:       true, // for now we default to true (accept misses; before duplicates)
		},
		TimestampField:  c.JSON.TimestampField,
		TimestampLayout: c.JSON.TimestampLayout,
		StatusField:     c.JSON.StatusField,
		AddressField:    c.JSON.AddressField,
		UserField:       c.JSON.UserField,
	}
}

func (s *Server) addAccesslogSensor(_ context.Context, config *AccesslogSensorConfig) (*sensor.Sensor, error) {
	s.logger.Info("adding sensor", slog.Any("sensor", config))
	var source accesslog.Sensor
	var err error
	switch config.Encoding {
	case LogEncodingRegexp:
		source, err = accesslog.ScanRegexp(config.Path, config.regexpScanOptions())
	case LogEncodingJSON:
		source, err = accesslog.ScanJSON(config.Path, config.jsonScanOptions())
	default:
		err = fmt.Errorf("unrecognized access_log encoding '%s'", config.Encoding)
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
