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
	"regexp"
)

type RegexpResolver struct {
	Pattern              *regexp.Regexp
	HostIndex            int
	HostValue            SymbolResolver
	TimestampIndex       int
	TimestampValue       TimestampResolver
	EventTypeIndex       int
	EventTypeValue       EventTypeResolver
	AddressIndex         int
	AddressValue         AddressResolver
	HardwareAddressIndex int
	HardwareAddressValue HardwareAddressResolver
	UserIndex            int
	UserValue            SymbolResolver
	ServiceIndex         int
	ServiceValue         SymbolResolver
}

func (r *RegexpResolver) Resolve(s string) (*ResolvedValues, bool) {
	match := r.Pattern.FindStringSubmatch(s)
	if match == nil {
		return nil, false
	}
	host, ok := r.HostValue.Resolve(r.symbol(match, r.HostIndex))
	if !ok {
		return nil, false
	}
	timestamp, ok := r.TimestampValue.Resolve(r.symbol(match, r.TimestampIndex))
	if !ok {
		return nil, false
	}
	eventType, ok := r.EventTypeValue.Resolve(r.symbol(match, r.EventTypeIndex))
	if !ok {
		return nil, false
	}
	address, ok := r.AddressValue.Resolve(r.symbol(match, r.AddressIndex))
	if !ok {
		return nil, false
	}
	hardwareAddress, ok := r.HardwareAddressValue.Resolve(r.symbol(match, r.HardwareAddressIndex))
	if !ok {
		return nil, false
	}
	user, ok := r.UserValue.Resolve(r.symbol(match, r.UserIndex))
	if !ok {
		return nil, false
	}
	service, ok := r.ServiceValue.Resolve(r.symbol(match, r.ServiceIndex))
	if !ok {
		return nil, false
	}
	resolved := &ResolvedValues{
		Host:            host,
		Timestamp:       timestamp,
		EventType:       eventType,
		Address:         address,
		HardwareAddress: hardwareAddress,
		User:            user,
		Service:         service,
	}
	return resolved, true
}

func (r *RegexpResolver) symbol(match []string, index int) string {
	if index < 0 {
		return ""
	}
	return match[index]
}
