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
	Timestamp       time.Time
	Type            EventType
	HardwareAddress net.HardwareAddr
	IPAddress       *netip.Addr
	User            string
	Service         string
	Source          string
}

func (e *Event) String() string {
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
	return fmt.Sprintf("timestamp:%s type:%s MAC:%s IP:%s User:%s Service:%s", timestamp, e.Type, mac, ip, user, service)
}

type EventReceiver interface {
	Queue(event *Event)
}

type EventReceiverFunc func(event *Event)

func (f EventReceiverFunc) Queue(event *Event) {
	f(event)
}

type EventSource interface {
	Collect(receiver EventReceiver) error
	Shutdown(ctx context.Context) error
	Close() error
}
