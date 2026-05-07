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

package dns

import (
	"context"
	"fmt"
	"net/netip"
)

type ProviderName string

type ProviderConfig interface {
	ProviderName() ProviderName
}

type Provider interface {
	Name() ProviderName
	Lookup(ctx context.Context, address netip.Addr) (string, error)
}

type OpenProviderFunc func(config ProviderConfig) (Provider, error)

var providers map[ProviderName]OpenProviderFunc = make(map[ProviderName]OpenProviderFunc)

func RegisterProvider(name ProviderName, open OpenProviderFunc) {
	providers[name] = open
}

func Open(config ProviderConfig) (Provider, error) {
	name := config.ProviderName()
	open, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown GeoIP provider name '%s'", name)
	}
	return open(config)
}
