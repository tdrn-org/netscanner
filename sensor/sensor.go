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

package sensor

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"time"
)

type EventType string

const (
	EventTypeGranted EventType = "granted"
	EventTypeDenied  EventType = "denied"
	EventTypeError   EventType = "error"
)

var eventTypeMap map[string]EventType = map[string]EventType{
	string(EventTypeGranted): EventTypeGranted,
	string(EventTypeDenied):  EventTypeDenied,
	string(EventTypeError):   EventTypeError,
}

func MatchEventType(s string) (EventType, bool) {
	eventType, match := eventTypeMap[s]
	return eventType, match
}

type Event struct {
	Host            string
	Timestamp       time.Time
	Type            EventType
	HardwareAddress net.HardwareAddr
	IPAddress       *netip.Addr
	User            string
	Service         string
	Sensor          string
}

func (e *Event) String() string {
	host := "-"
	if e.Host != "" {
		host = e.Host
	}
	timestamp := e.Timestamp.Format(time.RFC3339)
	mac := "-"
	if e.HardwareAddress != nil {
		mac = e.HardwareAddress.String()
	}
	ip := "-"
	if e.IPAddress != nil {
		ip = e.IPAddress.String()
	}
	user := "-"
	if e.User != "" {
		user = e.User
	}
	service := "-"
	if e.Service != "" {
		service = e.Service
	}
	return fmt.Sprintf("host:%s timestamp:%s type:%s MAC:%s IP:%s User:%s Service:%s", host, timestamp, e.Type, mac, ip, user, service)
}

type EventReceiver interface {
	Queue(event *Event)
}

type EventReceiverFunc func(event *Event)

func (f EventReceiverFunc) Queue(event *Event) {
	f(event)
}

type EventSource interface {
	Name() string
	Collect(receiver EventReceiver) error
	Shutdown(ctx context.Context) error
	Close() error
}

type Sensor struct {
	name   string
	source EventSource
}

var sensorNames map[string]int = make(map[string]int)
var sensorNamesMutex sync.Mutex = sync.Mutex{}

func New(name string, source EventSource) *Sensor {
	sensorNamesMutex.Lock()
	defer sensorNamesMutex.Unlock()

	baseName := fmt.Sprintf("%s/%s#", source.Name(), name)
	instance := sensorNames[baseName] + 1
	sensorNames[baseName] = instance
	return &Sensor{
		name:   baseName + strconv.Itoa(instance),
		source: source,
	}
}

func (s *Sensor) Name() string {
	return s.name
}

func (s *Sensor) Collect(receiver EventReceiver) error {
	return s.source.Collect(receiver)
}

func (s *Sensor) Shutdown(ctx context.Context) error {
	return s.source.Shutdown(ctx)
}

func (s *Sensor) Close() error {
	return s.source.Close()
}
