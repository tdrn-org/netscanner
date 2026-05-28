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

package netscanner

import (
	"fmt"
	"log/slog"
	"net/netip"

	"github.com/tdrn-org/netscanner/config"
	"github.com/tdrn-org/netscanner/internal/geoip"
	"github.com/tdrn-org/netscanner/internal/geoip/maxminddb"
)

type GeoIPConfig struct {
	Provider            GeoIPProvider `toml:"provider"`
	AddressURLTemplate  string        `toml:"address_url_template"`
	LocationURLTemplate string        `toml:"location_url_template"`
	Mappings            []struct {
		Networks config.NetworkSpecs `toml:"networks"`
		Host     string              `toml:"host"`
	} `toml:"mapping"`
	None      NoneGeoIPConfig      `toml:"none"`
	MaxMindDB MaxmindDBGeoIPConfig `toml:"maxminddb"`
}

func (c *GeoIPConfig) config() (geoip.ProviderConfig, map[netip.Prefix]string) {
	geoipMapping := make(map[netip.Prefix]string, 0)
	for _, configMapping := range c.Mappings {
		for _, network := range configMapping.Networks.Prefixes() {
			geoipMapping[network] = configMapping.Host
		}
	}
	switch c.Provider {
	case GeoIPProvider(geoip.NoneProviderName):
		return c.None.config(), geoipMapping
	case GeoIPProvider(maxminddb.ProviderName):
		return c.MaxMindDB.config(), geoipMapping
	}
	slog.Warn("unexpected GeoIP provider", slog.String("provider", string(c.Provider)))
	return nil, nil
}

type NoneGeoIPConfig struct {
	// no options
}

func (c *NoneGeoIPConfig) config() *geoip.None {
	return &geoip.None{}
}

type MaxmindDBGeoIPConfig struct {
	File string `toml:"file"`
}

func (c *MaxmindDBGeoIPConfig) config() *maxminddb.Config {
	return &maxminddb.Config{
		File: c.File,
	}
}

type GeoIPProvider geoip.ProviderName

var knownGeoIPProviders map[string]GeoIPProvider = map[string]GeoIPProvider{
	string(geoip.NoneProviderName): GeoIPProvider(geoip.NoneProviderName),
	string(maxminddb.ProviderName): GeoIPProvider(maxminddb.ProviderName),
}

func (p *GeoIPProvider) Value() string {
	for value, geoipProvider := range knownGeoIPProviders {
		if *p == geoipProvider {
			return value
		}
	}
	slog.Warn("unexpected GeoIP provider", slog.Any("p", *p))
	return ""
}

func (p *GeoIPProvider) MarshalTOML() ([]byte, error) {
	return []byte(`"` + p.Value() + `"`), nil
}

func (p *GeoIPProvider) UnmarshalTOML(value any) error {
	geoipProviderString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected GeoIP provider type %v", value)
	}
	geoipProvider, ok := knownGeoIPProviders[geoipProviderString]
	if !ok {
		return fmt.Errorf("unknown GeoIP provider: '%s'", geoipProviderString)
	}
	*p = geoipProvider
	return nil
}
