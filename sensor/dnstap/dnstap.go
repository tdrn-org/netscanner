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

package dnstap

import (
	"context"
	"time"

	"github.com/tdrn-org/netscanner/sensor"
)

const Name string = "dnstap"

type Sensor struct {
	path     string
	receiver Receiver
}

func ListenSocket(path string) (*Sensor, error) {
	receiver, err := NewSocketReceiver(path, 0666, DefaultMaxFrameSize, time.Now())
	if err != nil {
		return nil, err
	}
	sensor := &Sensor{
		path:     path,
		receiver: receiver,
	}
	return sensor, nil
}

func PollFile(path string) (*Sensor, error) {
	receiver, err := NewFileReceiver(path, DefaultMaxFrameSize, time.Now())
	if err != nil {
		return nil, err
	}
	sensor := &Sensor{
		path:     path,
		receiver: receiver,
	}
	return sensor, nil
}

func (s *Sensor) Path() string {
	return s.path
}

func (s *Sensor) Name() string {
	return Name
}

func (s *Sensor) Collect(receiver sensor.EventReceiver) error {
	s.receiver.Consume(func(entry *Entry) {
		client, server, ok := entry.Decode()
		if !ok {
			return
		}
		event := &sensor.Event{
			Host:      server.String(),
			Timestamp: time.Now(),
			Type:      sensor.EventTypeInformational,
			Address:   client,
			Service:   "dns",
			Sensor:    Name,
		}
		receiver.Queue(context.Background(), event)
	})
	return nil
}

func (s *Sensor) Shutdown(ctx context.Context) error {
	return s.receiver.Shutdown(ctx)
}

func (s *Sensor) Close() error {
	return s.receiver.Close()
}
