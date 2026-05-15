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
	"time"

	"github.com/tdrn-org/netscanner/sensor"
)

type Resolver[T any] interface {
	Resolve(s string) (T, bool)
}

type ResolverFunc[T any] func(s string) (T, bool)

func (f ResolverFunc[T]) Resolve(s string) (T, bool) {
	return f(s)
}

type TimestampResolver Resolver[time.Time]

type TimestampValueResolver struct {
	Layout string
}

func (r *TimestampValueResolver) Value(s string) (time.Time, bool) {
	value, err := time.Parse(r.Layout, s)
	return value, err == nil
}

type EventTypeResolver Resolver[sensor.EventType]

type AddressResolver Resolver[netip.Addr]

var AddressValueResolver ResolverFunc[netip.Addr] = func(s string) (netip.Addr, bool) {
	value, err := netip.ParseAddr(s)
	return value, err == nil
}

type HardwareAddressResolver Resolver[net.HardwareAddr]

var HardwareAddressValueResolver ResolverFunc[net.HardwareAddr] = func(s string) (net.HardwareAddr, bool) {
	value, err := net.ParseMAC(s)
	return value, err == nil
}

type SymbolResolver Resolver[string]

var SymbolValueResolver ResolverFunc[string] = func(s string) (string, bool) {
	return s, true
}

type StaticValueResolver[T any] struct {
	StaticValue T
}

func (r *StaticValueResolver[T]) Resolve(s string) (T, bool) {
	return r.StaticValue, true
}
