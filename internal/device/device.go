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
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/tdrn-org/netscanner/internal/arp"
	"github.com/tdrn-org/netscanner/internal/cache"
	"github.com/tdrn-org/netscanner/internal/cache/memory"
	"github.com/tdrn-org/netscanner/internal/dns"
	"github.com/tdrn-org/netscanner/internal/geoip"
	"github.com/tdrn-org/netscanner/internal/i18n"
	"github.com/tdrn-org/netscanner/network"
)

type Info struct {
	Address         netip.Addr
	Network         string
	HardwareAddress net.HardwareAddr
	DNS             string
	Geo             geoip.Info
}

func (i *Info) Equal(i2 *Info) bool {
	return i.Address == i2.Address && i.Network == i2.Network && i.DNS == i2.DNS && i.Geo.Equal(&i2.Geo)
}

func (i *Info) String() string {
	buffer := &strings.Builder{}
	buffer.WriteString("Address:")
	buffer.WriteString(i.Address.String())
	buffer.WriteString(" Network:")
	buffer.WriteString(i.Network)
	if i.HardwareAddress != nil {
		buffer.WriteString(" MAC:")
		buffer.WriteString(i.HardwareAddress.String())
	}
	if i.DNS != "" {
		buffer.WriteString(" DNS:")
		buffer.WriteString(i.DNS)
	}
	if !i.Geo.IsNaN() {
		buffer.WriteString(" Loc:")
		buffer.WriteString(strconv.FormatFloat(i.Geo.Lat, 'f', 2, 64))
		buffer.WriteString(",")
		buffer.WriteString(strconv.FormatFloat(i.Geo.Lng, 'f', 2, 64))
	}
	if len(i.Geo.City) > 0 {
		buffer.WriteString(" City:")
		buffer.WriteString(i18n.Name(i.Geo.City).String())
	}
	if len(i.Geo.Country) > 0 {
		buffer.WriteString(" Country:")
		buffer.WriteString(i18n.Name(i.Geo.Country).String())
	}
	return buffer.String()
}

type InfoCache struct {
	networks      *network.Names
	arpCache      *arp.Cache
	dnsProvider   dns.Provider
	dnsDomains    []string
	geoipProvider geoip.Provider
	infoCache     cache.KeyValue[netip.Addr, *Info]
	hostCache     cache.KeyValue[string, []netip.Addr]
}

func NewInfoCache(networks *network.Names, arpCache *arp.Cache, dnsProvider dns.Provider, dnsDomains []string, geoipProvider geoip.Provider) (*InfoCache, error) {
	c := &InfoCache{
		networks:      networks,
		arpCache:      arpCache,
		dnsProvider:   dnsProvider,
		geoipProvider: geoipProvider,
	}
	ttl := time.Hour
	infoCache, err := memory.NewKeyValue(0, ttl, c.loadInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create device info cache (cause: %w)", err)
	}
	hostCache, err := memory.NewKeyValue(0, ttl, c.loadHost)
	c.infoCache = infoCache
	c.hostCache = hostCache
	return c, nil
}

func (c *InfoCache) loadInfo(ctx context.Context, address netip.Addr) (*Info, error) {
	addressLogger := slog.With(slog.String("address", address.String()))
	hostName, err := c.dnsProvider.LookupAddress(ctx, address)
	if err != nil && !errors.Is(err, dns.ErrNotFound) {
		addressLogger.Warn("failed to query DNS info", slog.Any("err", err))
	}
	info := &Info{
		Address: address,
		DNS:     hostName,
		Geo:     *geoip.NoInfo,
	}
	info.Network = c.networks.Match(address)
	info.HardwareAddress = c.arpCache.Get(ctx, address)
	geoipInfo, err := c.geoipProvider.Lookup(ctx, address)
	if err == nil {
		info.Geo = *geoipInfo
	} else {
		addressLogger.Warn("failed to query GeoIP info", slog.Any("err", err))
	}
	return info, nil
}

func (c *InfoCache) loadHost(ctx context.Context, host string) ([]netip.Addr, error) {
	hostLogger := slog.With(slog.String("host", host))
	hostNames := make([]string, 0, len(c.dnsDomains)+1)
	hostNames = append(hostNames, host)
	if !strings.HasSuffix(host, ".") {
		for _, dnsDomain := range c.dnsDomains {
			hostNames = append(hostNames, host+"."+dnsDomain)
		}
	}
	addrs := make([]netip.Addr, 0)
	for _, hostName := range hostNames {
		addr, err := c.dnsProvider.LookupHost(ctx, hostName)
		if err != nil {
			if !errors.Is(err, dns.ErrNotFound) {
				hostLogger.Warn("failed to lookup host", slog.String("host", hostName), slog.Any("err", err))
			}
			continue
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

func (c *InfoCache) LookupHost(ctx context.Context, host string, clientAddress netip.Addr) (*Info, bool) {
	hostAddrs, hit := c.hostCache.Get(ctx, host)
	if !hit {
		return nil, false
	}
	for _, hostAddr := range hostAddrs {
		if hostAddr.Is4() && clientAddress.Is4() {
			if c.matchAddrClass(hostAddr, clientAddress) {
				return c.LookupAddress(ctx, hostAddr)
			}
		} else if c.matchAddrClass(hostAddr, clientAddress) {
			return c.LookupAddress(ctx, hostAddr)
		}
	}
	return nil, false
}

func (c *InfoCache) matchAddrClass(addr1 netip.Addr, addr2 netip.Addr) bool {
	if addr1.IsLoopback() {
		return addr2.IsLoopback()
	} else if addr1.IsPrivate() {
		return addr2.IsPrivate()
	}
	return true
}

func (c *InfoCache) LookupAddress(ctx context.Context, address netip.Addr) (*Info, bool) {
	return c.infoCache.Get(ctx, address)
}

func (c *InfoCache) Close() error {
	return c.geoipProvider.Close()
}
