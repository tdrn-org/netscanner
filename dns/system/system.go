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

package system

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"

	"github.com/tdrn-org/netscanner/dns"
)

const ProviderName dns.ProviderName = "system"

type Config struct {
	// No options
}

func (c *Config) ProviderName() dns.ProviderName {
	return ProviderName
}

type systemProvider struct {
	config   *Config
	resolver *net.Resolver
	logger   *slog.Logger
}

func open(config dns.ProviderConfig) (dns.Provider, error) {
	systemConfig, ok := config.(*Config)
	if !ok {
		return nil, fmt.Errorf("not a System DNS configuration")
	}
	provider := &systemProvider{
		config:   systemConfig,
		resolver: net.DefaultResolver,
		logger:   slog.With(slog.String("dns", string(systemConfig.ProviderName()))),
	}
	return provider, nil
}

func (p *systemProvider) Name() dns.ProviderName {
	return p.config.ProviderName()
}

func (p *systemProvider) LookupHost(ctx context.Context, host string) (netip.Addr, error) {
	hostLogger := p.logger.With(slog.String("host", host))
	hostLogger.Debug("Looking up host...")
	lookupResult, err := p.resolver.LookupHost(ctx, host)
	if err != nil {
		hostLogger.Warn("DNS lookup failure", slog.Any("err", err))
	}
	if len(lookupResult) == 0 {
		return netip.Addr{}, dns.ErrNotFound
	}
	// Sort results to ensure stable result in case there are multiple returned
	slices.Sort(lookupResult)
	lookupResult0 := lookupResult[0]
	address, err := netip.ParseAddr(lookupResult0)
	if err != nil {
		hostLogger.Error("invalid address in  DNS result", slog.Any("address", lookupResult0), slog.Any("err", err))
		return netip.Addr{}, dns.ErrNotFound
	}
	hostLogger.Debug("Looked up host", slog.String("address", address.String()))
	return address, nil
}

func (p *systemProvider) LookupAddress(ctx context.Context, address netip.Addr) (string, error) {
	addressString := address.String()
	addressLogger := p.logger.With(slog.String("address", addressString))
	addressLogger.Debug("Looking up address...")
	lookupResult, err := p.resolver.LookupAddr(ctx, addressString)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); !ok || !dnsErr.IsNotFound {
			addressLogger.Warn("DNS reverse lookup failure", slog.Any("err", err))
		}
	}
	if len(lookupResult) == 0 {
		return "", dns.ErrNotFound
	}
	// Sort results to ensure stable result in case there are multiple returned
	slices.Sort(lookupResult)
	lookupResult0 := lookupResult[0]
	addressLogger.Debug("Looked up address", slog.String("name", lookupResult0))
	return lookupResult0, nil
}

func init() {
	dns.RegisterProvider(ProviderName, open)
}
