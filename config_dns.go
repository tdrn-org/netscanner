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

	"github.com/tdrn-org/netscanner/dns"
	customdns "github.com/tdrn-org/netscanner/dns/custom"
	systemdns "github.com/tdrn-org/netscanner/dns/system"
)

type DNSConfig struct {
	Provider  DNSProvider     `toml:"provider"`
	Domains   []string        `toml:"domains"`
	SystemDNS SystemDNSConfig `toml:"system"`
	CustomDNS CustomDNSConfig `toml:"custom"`
}

func (c *DNSConfig) config() dns.ProviderConfig {
	switch c.Provider {
	case DNSProvider(systemdns.ProviderName):
		return c.SystemDNS.config()
	case DNSProvider(customdns.ProviderName):
		return c.CustomDNS.config()
	}
	slog.Warn("unexpected DNS provider", slog.String("provider", string(c.Provider)))
	return nil
}

type SystemDNSConfig struct {
	// no options
}

func (c *SystemDNSConfig) config() *systemdns.Config {
	return &systemdns.Config{}
}

type CustomDNSConfig struct {
	Network string `toml:"network"`
	Address string `toml:"address"`
}

func (c *CustomDNSConfig) config() *customdns.Config {
	return &customdns.Config{
		Network: c.Network,
		Address: c.Address,
	}
}

type DNSProvider dns.ProviderName

var knownDNSProviders map[string]DNSProvider = map[string]DNSProvider{
	string(systemdns.ProviderName): DNSProvider(systemdns.ProviderName),
	string(customdns.ProviderName): DNSProvider(customdns.ProviderName),
}

func (p *DNSProvider) Value() string {
	for value, dnsProvider := range knownDNSProviders {
		if *p == dnsProvider {
			return value
		}
	}
	slog.Warn("unexpected DNS provider", slog.Any("p", *p))
	return ""
}

func (p *DNSProvider) MarshalTOML() ([]byte, error) {
	return []byte(`"` + p.Value() + `"`), nil
}

func (p *DNSProvider) UnmarshalTOML(value any) error {
	dnsProviderString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected DNS provider type %v", value)
	}
	dnsProvider, ok := knownDNSProviders[dnsProviderString]
	if !ok {
		return fmt.Errorf("unknown DNS provider: '%s'", dnsProviderString)
	}
	*p = dnsProvider
	return nil
}
