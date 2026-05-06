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

type Provider interface {
	Lookup(ctx context.Context, address netip.Addr) (string, error)
}

type resolverProvider struct {
	resolver *net.Resolver
	logger   *slog.Logger
}

func NewResolverProvider(resolver *net.Resolver) Provider {
	return &resolverProvider{
		resolver: resolver,
		logger:   slog.With(slog.String("dns", "resolver")),
	}
}

func (p *resolverProvider) Lookup(ctx context.Context, address netip.Addr) (string, error) {
	addressString := address.String()
	addressLogger := p.logger.With(slog.String("address", addressString))
	addressLogger.Debug("Looking up host name...")
	names, err := p.resolver.LookupAddr(ctx, addressString)
	if err != nil {
		addressLogger.Info("DNS lookup failure", slog.Any("err", err))
	}
	if len(names) == 0 {
		return "", nil
	}
	// Sort names to ensure stable result in case there are multiple names
	slices.Sort(names)
	name := names[0]
	addressLogger.Debug("Looked up host name", slog.String("name", name))
	return name, nil
}
