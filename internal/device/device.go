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
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/netip"
	"time"

	"github.com/tdrn-org/netscanner/arp"
	"github.com/tdrn-org/netscanner/dns"
	"github.com/tdrn-org/netscanner/geoip"
	"github.com/tdrn-org/netscanner/internal/cache"
	"github.com/tdrn-org/netscanner/internal/cache/memory"
	"github.com/tdrn-org/netscanner/network"
	"golang.org/x/text/language"
)

type Info struct {
	Address         netip.Addr
	Network         string
	HardwareAddress net.HardwareAddr
	DNS             string
	Geoip           geoip.Info
}

func (i *Info) Equal(i2 *Info) bool {
	return i.Address == i2.Address && i.Network == i2.Network && i.DNS == i2.DNS && i.Geoip.Equal(&i2.Geoip)
}

func (i *Info) String() string {
	dns := "-"
	if i.DNS != "" {
		dns = i.DNS
	}
	return fmt.Sprintf("Address:%s Network:%s DNS:%s", i.Address, i.Network, dns)
}

type InfoCache struct {
	networks *network.Names
	arp      arp.Provider
	dns      dns.Provider
	geoip    geoip.Provider
	cache    cache.KeyValue[netip.Addr, *Info]
}

func NewInfoCache(networks *network.Names, arp arp.Provider, dns dns.Provider, geoip geoip.Provider) (*InfoCache, error) {
	c := &InfoCache{
		networks: networks,
		arp:      arp,
		dns:      dns,
		geoip:    geoip,
	}
	// TODO: Cache configuration
	cache, err := memory.NewKeyValue(0, time.Hour, c.loadInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create device info cache (cause: %w)", err)
	}
	c.cache = cache
	return c, nil
}

func (c *InfoCache) loadInfo(ctx context.Context, address netip.Addr) (*Info, error) {
	logger := slog.With(slog.String("addr", address.String()))
	info := &Info{
		Address: address,
		Geoip: geoip.Info{
			Lat:     math.NaN(),
			Lng:     math.NaN(),
			City:    map[language.Tag]string{},
			Country: map[language.Tag]string{},
		},
	}
	info.Network = c.networks.Match(address)
	info.HardwareAddress = c.arp.Lookup(ctx, address)
	dns, err := c.dns.Lookup(ctx, address)
	if err == nil {
		info.DNS = dns
	} else {
		logger.Warn("failed to query DNS info", slog.Any("err", err))
	}
	geoipInfo, err := c.geoip.Lookup(ctx, address)
	if err == nil {
		info.Geoip = *geoipInfo
	} else {
		logger.Warn("failed to query GeoIP info", slog.Any("err", err))
	}
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
	return errors.Join(c.arp.Close(), c.geoip.Close())
}
