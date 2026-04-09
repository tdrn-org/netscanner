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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tdrn-org/netscanner/sensor"
)

type Recorder interface {
	RecordEvent(event *sensor.Event)
}

func NewRecorder(registry *prometheus.Registry) Recorder {
	if registry == nil {
		return &noopRecorder{}
	}
	return newMetricsRecorder(registry)
}

type noopRecorder struct{}

func (r *noopRecorder) RecordEvent(_ *sensor.Event) {
	// no-op
}

type metricsRecorder struct {
	Events *prometheus.CounterVec
}

const metricsNamespace string = "netscanner"
const metricsSubsystemSensors string = "sensors"

func newMetricsRecorder(registry *prometheus.Registry) *metricsRecorder {
	factory := promauto.With(registry)
	recorder := &metricsRecorder{
		Events: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystemSensors,
			Name:      "events",
		}, []string{"sensor", "service", "type"}),
	}
	return recorder
}

func (r *metricsRecorder) RecordEvent(event *sensor.Event) {
	//TODO:Sensor
	r.Events.WithLabelValues("syslog", event.Service, string(event.Type)).Inc()
}
