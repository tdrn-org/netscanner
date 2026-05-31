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

	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/dnstap"
)

func (s *Server) addDnstapSensor(ctx context.Context, config *DnstapSensorConfig) (*sensor.Sensor, error) {
	s.logger.Info("adding dnstap sensor", "name", config.Name, "file", config.File)

	source, err := dnstap.NewFileSource(config.File)
	if err != nil {
		return nil, fmt.Errorf("failed to create dnstap sensor %q: %w", config.Name, err)
	}

	ss := sensor.New(config.Name, source)
	s.mutex.Lock()
	s.sensors[ss.Name()] = ss
	s.mutex.Unlock()

	return ss, nil
}
