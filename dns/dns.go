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
	"log/slog"
	"net"
	"net/netip"
	"slices"
)

type Info struct {
	Name string
}

func (i *Info) Equal(i2 *Info) bool {
	return i.Name == i2.Name
}

type Provider interface {
	Lookup(ctx context.Context, address netip.Addr) (*Info, error)
}

type resolverProvider struct {
	resolver *net.Resolver
	mapper   func(netip.Addr) netip.Addr
}

func NewResolverProvider(resolver *net.Resolver, mapper func(netip.Addr) netip.Addr) Provider {
	return &resolverProvider{
		resolver: resolver,
		mapper:   mapper,
	}
}

func (p *resolverProvider) Lookup(ctx context.Context, address netip.Addr) (*Info, error) {
	mappedAddress := address
	if p.mapper != nil {
		mappedAddress = p.mapper(address)
	}
	addressString := mappedAddress.String()
	names, err := p.resolver.LookupAddr(ctx, addressString)
	if err != nil {
		slog.Info("DNS lookup failure", slog.String("address", addressString), slog.Any("err", err))
	}
	if len(names) == 0 {
		info := &Info{
			Name: addressString,
		}
		return info, nil
	}
	// Sort names to ensure stable result in case there are multiple names
	slices.Sort(names)
	info := &Info{
		Name: names[0],
	}
	return info, nil
}
