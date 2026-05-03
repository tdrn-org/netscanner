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

package logmatcher

import (
	"net"
	"net/netip"
	"strings"

	"github.com/tdrn-org/netscanner/sensor"
)

type Value string

const (
	AnyValue             Value = ""
	AddressValue         Value = "\x01"
	HardwareAddressValue Value = "\x02"
	UserValue            Value = "\x03"
	ServiceValue         Value = "\x04"
)

func ParseValue(s string) Value {
	switch s {
	case "{Any}":
		return AnyValue
	case "{IP}":
		return AddressValue
	case "{MAC}":
		return HardwareAddressValue
	case "{User}":
		return UserValue
	case "{Service}":
		return ServiceValue
	default:
		return Value(strings.ReplaceAll(strings.ReplaceAll(s, "{{", "{"), "}}", "}"))
	}
}

func (value Value) String() string {
	switch value {
	case AnyValue:
		return "{Any}"
	case AddressValue:
		return "{IP}"
	case HardwareAddressValue:
		return "{MAC}"
	case UserValue:
		return "{User}"
	case ServiceValue:
		return "{Service}"
	default:
		return strings.ReplaceAll(strings.ReplaceAll(string(value), "{", "{{"), "}", "}}")
	}
}

type Match []Value

func ParseMatch(s string) Match {
	valueStrings := strings.Split(s, " ")
	match := make(Match, len(valueStrings))
	for i, valueString := range valueStrings {
		match[i] = ParseValue(valueString)
	}
	return match
}

func (match Match) String() string {
	buffer := &strings.Builder{}
	for _, value := range match {
		if buffer.Len() > 0 {
			buffer.WriteRune(' ')
		}
		buffer.WriteString(value.String())
	}
	return buffer.String()
}

type ResolvedValues struct {
	EventType       sensor.EventType
	Address         netip.Addr
	HardwareAddress net.HardwareAddr
	User            string
	Service         string
}
