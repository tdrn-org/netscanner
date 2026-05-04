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

package device

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/tdrn-org/netscanner/dns"
	"github.com/tdrn-org/netscanner/internal/cache"
	"github.com/tdrn-org/netscanner/internal/cache/memory"
	"github.com/tdrn-org/netscanner/network"
)

type Info struct {
	Address netip.Addr
	Network string
	DNS     dns.Info
}

func (i *Info) Equal(i2 *Info) bool {
	return i.Address == i2.Address && i.Network == i2.Network && i.DNS.Equal(&i2.DNS)
}

func (i *Info) String() string {
	dnsName := "-"
	if i.DNS.Name != "" {
		dnsName = i.DNS.Name
	}
	return fmt.Sprintf("Address:%s Network:%s DNS:%s", i.Address, i.Network, dnsName)
}

type InfoCache struct {
	dns      dns.Provider
	networks *network.Names
	cache    cache.KeyValue[netip.Addr, *Info]
}

func NewInfoCache(networks *network.Names, dns dns.Provider) (*InfoCache, error) {
	c := &InfoCache{
		networks: networks,
		dns:      dns,
	}
	// TODO: Cache configuration
	cache, err := memory.NewKeyValue(0, time.Hour, c.loadInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache (cause: %w)", err)
	}
	c.cache = cache
	return c, nil
}

func (c *InfoCache) loadInfo(ctx context.Context, address netip.Addr) (*Info, error) {
	logger := slog.With(slog.String("addr", address.String()))
	info := &Info{
		Address: address,
	}
	dnsInfo, err := c.dns.Lookup(ctx, address)
	if err == nil {
		info.DNS = *dnsInfo
	} else {
		logger.Warn("failed to query DNS info", slog.Any("err", err))
	}
	info.Network = c.networks.Match(address)
	return info, nil
}

func (c *InfoCache) Lookup(ctx context.Context, address netip.Addr) *Info {
	deviceInfo, match := c.cache.Get(ctx, address)
	if !match {
		return &Info{Address: address}
	}
	return deviceInfo
}

func (c *InfoCache) Close() error {
	return nil
}
