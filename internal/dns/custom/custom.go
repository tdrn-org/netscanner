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

package custom

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"

	customdns "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/tdrn-org/netscanner/internal/dns"
)

const ProviderName dns.ProviderName = "custom"

type Config struct {
	Network string
	Address string
}

func (c *Config) ProviderName() dns.ProviderName {
	return ProviderName
}

type customProvider struct {
	config *Config
	client *customdns.Client
	logger *slog.Logger
}

func open(config dns.ProviderConfig) (dns.Provider, error) {
	customConfig, ok := config.(*Config)
	if !ok {
		return nil, fmt.Errorf("not a Custom DNS configuration")
	}
	p := &customProvider{
		config: customConfig,
		client: customdns.NewClient(),
		logger: slog.With(slog.String("dns", fmt.Sprintf("%s://%s", customConfig.Network, customConfig.Address))),
	}
	return p, nil
}

func (p *customProvider) Name() dns.ProviderName {
	return p.config.ProviderName()
}

func (p *customProvider) LookupHost(ctx context.Context, host string) (netip.Addr, error) {
	hostLogger := p.logger.With(slog.String("host", host))
	hostLogger.Debug("Looking up host...")
	fdqn := dnsutil.Fqdn(host)
	records, err := p.lookupHost(ctx, fdqn, customdns.TypeA)
	if err == nil && len(records) == 0 {
		records, err = p.lookupHost(ctx, fdqn, customdns.TypeAAAA)
	}
	if err != nil {
		hostLogger.Warn("DNS lookup failure", slog.Any("err", err))
		return netip.Addr{}, fmt.Errorf("%w (cause: %w)", dns.ErrNotFound, err)
	}
	if len(records) == 0 {
		return netip.Addr{}, dns.ErrNotFound
	}
	// Sort results to ensure stable result in case there are multiple returned
	slices.SortFunc(records, func(a1, a2 netip.Addr) int { return a1.Compare(a2) })
	record0 := records[0]
	hostLogger.Debug("Looked up address", slog.String("name", record0.String()))
	return record0, nil
}

func (p *customProvider) lookupHost(ctx context.Context, fdqn string, t uint16) ([]netip.Addr, error) {
	msg := customdns.NewMsg(fdqn, t)
	rsp, _, err := p.client.Exchange(ctx, msg, p.config.Network, p.config.Address)
	if err != nil {
		return nil, err
	}
	records := make([]netip.Addr, 0, len(rsp.Answer))
	for _, answer := range rsp.Answer {
		if aRecord, ok := answer.(*customdns.A); ok {
			records = append(records, aRecord.Addr)
		} else if aaaaRecord, ok := answer.(*customdns.AAAA); ok {
			records = append(records, aaaaRecord.Addr)
		}
	}
	return records, nil
}

func (p *customProvider) LookupAddress(ctx context.Context, address netip.Addr) (string, error) {
	addressString := address.String()
	addressLogger := p.logger.With(slog.String("address", addressString))
	addressLogger.Debug("Looking up address...")
	ptr := dnsutil.ReverseAddr(address)
	msg := customdns.NewMsg(ptr, customdns.TypePTR)
	rsp, _, err := p.client.Exchange(ctx, msg, p.config.Network, p.config.Address)
	if err != nil {
		addressLogger.Warn("DNS lookup failure", slog.Any("err", err))
		return "", fmt.Errorf("%w (cause: %w)", dns.ErrNotFound, err)
	}
	records := make([]string, 0, len(rsp.Answer))
	for _, answer := range rsp.Answer {
		if ptrRecord, ok := answer.(*customdns.PTR); ok {
			records = append(records, ptrRecord.Ptr)
		}
	}
	if len(records) == 0 {
		return "", dns.ErrNotFound
	}
	// Sort results to ensure stable result in case there are multiple returned
	slices.Sort(records)
	record0 := records[0]
	addressLogger.Debug("Looked up address", slog.String("name", record0))
	return record0, nil
}

func init() {
	dns.RegisterProvider(ProviderName, open)
}
