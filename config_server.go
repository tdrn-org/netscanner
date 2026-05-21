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

	"github.com/rs/cors"
	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-httpserver/certificate"
	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/netscanner/config"
)

type ServerConfig struct {
	Address            string              `toml:"address"`
	Protocol           ServerProtocol      `toml:"protocol"`
	CertFile           string              `toml:"cert_file"`
	KeyFile            string              `toml:"key_file"`
	PublicURL          config.URLSpec      `toml:"public_url"`
	TrustedProxies     config.NetworkSpecs `toml:"trusted_proxies"`
	TrustedHeaders     []string            `toml:"trusted_headers"`
	AllowedOrigins     []string            `toml:"allowed_origins"`
	AccessLog          string              `toml:"access_log"`
	AccessLogSizeLimit int64               `toml:"access_log_size_limit"`
}

func (c *ServerConfig) httpServerOptions() []httpserver.OptionSetter {
	httpServerOptions := make([]httpserver.OptionSetter, 0)
	// TLS
	if c.Protocol == ServerProtocolHttps {
		certificateProvider := &certificate.FileCertificateProvider{
			CertFile: c.CertFile,
			KeyFile:  c.KeyFile,
		}
		httpServerOptions = append(httpServerOptions, httpserver.WithCertificateProvider(certificateProvider))
	}
	// Proxy configuration
	if len(c.TrustedProxies) > 0 {
		httpServerOptions = append(httpServerOptions, httpserver.WithTrustedProxyPolicy(httpserver.AllowNetworks("trusted proxies", c.TrustedProxies.Prefixes())))
	}
	if len(c.TrustedHeaders) > 0 {
		httpServerOptions = append(httpServerOptions, httpserver.WithTrustedHeaders(c.TrustedHeaders...))
	}
	// CORS
	if len(c.AllowedOrigins) > 0 {
		corsOptions := &cors.Options{
			AllowedOrigins: c.AllowedOrigins,
		}
		httpServerOptions = append(httpServerOptions, httpserver.WithCorsOptions(corsOptions))
	}
	// Access log
	var accessLogConfig *log.Config
	switch c.AccessLog {
	case "stdout":
		accessLogConfig = &log.Config{
			Target: log.TargetStdout,
		}
	case "stderr":
		accessLogConfig = &log.Config{
			Target: log.TargetStderr,
		}
	case "":
		// disable Access log
	default:
		accessLogConfig = &log.Config{
			Target:        log.TargetFileText,
			FileName:      c.AccessLog,
			FileSizeLimit: c.AccessLogSizeLimit,
		}
	}
	if accessLogConfig != nil {
		accessLogLogger := slog.New(log.NewRawHandler(accessLogConfig.GetWriter()))
		httpServerOptions = append(httpServerOptions, httpserver.WithAccessLog(accessLogLogger))
	}
	return httpServerOptions
}

type ServerProtocol string

const (
	ServerProtocolHttp  ServerProtocol = "http"
	ServerProtocolHttps ServerProtocol = "https"
)

var knownServerProtocols map[string]ServerProtocol = map[string]ServerProtocol{
	string(ServerProtocolHttp):  ServerProtocolHttp,
	string(ServerProtocolHttps): ServerProtocolHttps,
}

func (p *ServerProtocol) Value() string {
	for value, protocol := range knownServerProtocols {
		if *p == protocol {
			return value
		}
	}
	slog.Warn("unexpected server protocol", slog.Any("p", *p))
	return ""
}

func (p *ServerProtocol) MarshalTOML() ([]byte, error) {
	return []byte(`"` + p.Value() + `"`), nil
}

func (p *ServerProtocol) UnmarshalTOML(value any) error {
	protocolString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected server protocol type %v", value)
	}
	protocol, ok := knownServerProtocols[protocolString]
	if !ok {
		return fmt.Errorf("unknown log target: '%s'", protocolString)
	}
	*p = protocol
	return nil
}
