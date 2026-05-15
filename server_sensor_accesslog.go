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
	"github.com/tdrn-org/netscanner/sensor/accesslog"
)

func (s *Server) addAccesslogSensor(ctx context.Context, config *AccesslogSensorConfig) (*sensor.Sensor, error) {
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
