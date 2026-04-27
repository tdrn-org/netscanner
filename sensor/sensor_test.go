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

package sensor_test

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/sensor"
)

func TestEventString(t *testing.T) {
	event := newTestEvent(t)
	fmt.Println(event.String())
}

func TestNewSensor(t *testing.T) {
	sensor := sensor.New("test", newTestSource(t))
	fmt.Println(sensor.Name())
}

func newTestEvent(t *testing.T) *sensor.Event {
	mac, err := net.ParseMAC("00:00:00:00:00:00")
	require.NoError(t, err)
	ip := netip.IPv4Unspecified()
	event := &sensor.Event{
		Timestamp:       time.Now(),
		Type:            sensor.EventTypeGranted,
		HardwareAddress: mac,
		IPAddress:       &ip,
		User:            "<user>",
		Service:         "<service>",
		Sensor:          "<sensor>",
	}
	return event
}

type testSensor struct {
	event  *sensor.Event
	ctx    context.Context
	cancel context.CancelFunc
}

func newTestSource(t *testing.T) sensor.EventSource {
	ctx, cancel := context.WithCancel(t.Context())
	source := &testSensor{
		event:  newTestEvent(t),
		ctx:    ctx,
		cancel: cancel,
	}
	return source
}

func (s *testSensor) Name() string {
	return "test"
}

func (s *testSensor) Collect(receiver sensor.EventReceiver) error {
	for {
		select {
		case <-time.After(time.Second):
			event := *s.event
			event.Timestamp = time.Now()
			receiver.Queue(&event)
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *testSensor) Shutdown(ctx context.Context) error {
	s.cancel()
	return nil
}

func (s *testSensor) Close() error {
	return nil
}
