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
	"github.com/tdrn-org/netscanner/internal/device"
	"github.com/tdrn-org/netscanner/sensor"
)

type Recorder interface {
	RecordEvent(event *sensor.Event, deviceInfo *device.Info)
}

func NewRecorder(registry *prometheus.Registry) Recorder {
	if registry == nil {
		return &noopRecorder{}
	}
	return newMetricsRecorder(registry)
}

type noopRecorder struct{}

func (r *noopRecorder) RecordEvent(_ *sensor.Event, _ *device.Info) {
	// no-op
}

type metricsRecorder struct {
	Events       *prometheus.CounterVec
	Geos         *prometheus.CounterVec
	GeoHashChars uint
}

const metricsNamespace string = "netscanner"
const metricsSubsystemSensors string = "sensors"
const metricsNameEvents string = "events"
const metricsNameGeos string = "geos"

func newMetricsRecorder(registry *prometheus.Registry) *metricsRecorder {
	factory := promauto.With(registry)
	recorder := &metricsRecorder{
		Events: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystemSensors,
			Name:      metricsNameEvents,
		}, []string{"host", "service", "network", "type"}),
		Geos: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystemSensors,
			Name:      metricsNameGeos,
		}, []string{"host", "service", "network", "type", "geohash"}),
		GeoHashChars: 12,
	}
	return recorder
}

func (r *metricsRecorder) RecordEvent(event *sensor.Event, deviceInfo *device.Info) {
	eventsCounter, err := r.Events.GetMetricWithLabelValues(event.Host, event.Service, deviceInfo.Network, string(event.Type))
	if err == nil {
		eventsCounter.Inc()
	}
	if !deviceInfo.Geo.IsNaN() {
		geosCounter, err := r.Geos.GetMetricWithLabelValues(event.Host, event.Service, deviceInfo.Network, string(event.Type), deviceInfo.Geo.Hash(r.GeoHashChars))
		if err != nil {
			geosCounter.Inc()
		}
	}
}
