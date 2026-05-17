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
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/tdrn-org/netscanner/dns"
	"github.com/tdrn-org/netscanner/geoip"
	"github.com/tdrn-org/netscanner/internal/arp"
	"github.com/tdrn-org/netscanner/internal/cache"
	"github.com/tdrn-org/netscanner/internal/cache/memory"
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
	cache         cache.KeyValue[string, *Info]
}

func NewInfoCache(networks *network.Names, arpCache *arp.Cache, dnsProvider dns.Provider, dnsDomains []string, geoipProvider geoip.Provider) (*InfoCache, error) {
	c := &InfoCache{
		networks:      networks,
		arpCache:      arpCache,
		dnsProvider:   dnsProvider,
		geoipProvider: geoipProvider,
	}
	cache, err := memory.NewKeyValue(0, time.Hour, c.loadInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create device info cache (cause: %w)", err)
	}
	c.cache = cache
	return c, nil
}

func (c *InfoCache) loadInfo(ctx context.Context, query string) (*Info, error) {
	queryLogger := slog.With(slog.String("query", query))
	address, err := netip.ParseAddr(query)
	hostName := ""
	if err != nil {
		hostName, err = c.dnsProvider.LookupAddress(ctx, address)
		if err != nil {
			queryLogger.Warn("failed to query DNS info", slog.Any("err", err))
		}
	} else {
		address, hostName, err = c.lookupQuery(ctx, query)
		if err != nil {
			return nil, err
		}
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
		queryLogger.Warn("failed to query GeoIP info", slog.Any("err", err))
	}
	return info, nil
}

func (c *InfoCache) lookupQuery(ctx context.Context, query string) (netip.Addr, string, error) {
	hostNames := make([]string, 0, len(c.dnsDomains)+1)
	hostNames = append(hostNames, query)
	if !strings.HasSuffix(query, ".") {
		for _, dnsDomain := range c.dnsDomains {
			hostNames = append(hostNames, query+"."+dnsDomain)
		}
	}
	for _, hostName := range hostNames {
		addr, err := c.dnsProvider.LookupHost(ctx, hostName)
		if err != nil {
			continue
		}
		return addr, hostName, nil
	}
	return netip.Addr{}, "", cache.ErrNotFound
}

func (c *InfoCache) Lookup(ctx context.Context, query string) (*Info, bool) {
	return c.cache.Get(ctx, query)
}

func (c *InfoCache) Close() error {
	return c.geoipProvider.Close()
}
