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
	_ "embed"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/rs/cors"
	"github.com/tdrn-org/go-conf/service/loglevel"
	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-httpserver/certificate"
	"github.com/tdrn-org/go-log"
)

type Config struct {
	Logging LoggingConfig `toml:"logging"`
	Server  ServerConfig  `toml:"server"`
	Metrics MetricsConfig `toml:"metrics"`
}

type LoggingConfig struct {
	Level          LogLevel  `toml:"level"`
	Target         LogTarget `toml:"target"`
	Color          LogColor  `toml:"color"`
	FileName       string    `toml:"file_name"`
	FileSizeLimit  int64     `toml:"file_size_limit"`
	SyslogNetwork  string    `toml:"syslog_network"`
	SyslogAddress  string    `toml:"syslog_address"`
	SyslogEncoding string    `toml:"syslog_encoding"`
	SyslogFacility int       `toml:"syslog_facility"`
}

func (c *LoggingConfig) apply() {
	logConfig := &log.Config{
		Level:          c.Level.Value(),
		AddSource:      false,
		Target:         log.Target(c.Target),
		Color:          log.Color(c.Color),
		FileName:       c.FileName,
		FileSizeLimit:  c.FileSizeLimit,
		SyslogNetwork:  c.SyslogNetwork,
		SyslogAddress:  c.SyslogAddress,
		SyslogEncoding: c.SyslogEncoding,
		SyslogFacility: c.SyslogFacility,
		SyslogAppName:  reflect.TypeFor[Server]().PkgPath(),
	}
	logger, _ := logConfig.GetLogger(loglevel.LevelVar())
	slog.SetDefault(logger)
}

type ServerConfig struct {
	Address            string         `toml:"address"`
	Protocol           ServerProtocol `toml:"protocol"`
	CertFile           string         `toml:"cert_file"`
	KeyFile            string         `toml:"key_file"`
	PublicURL          URLSpec        `toml:"public_url"`
	TrustedProxies     NetworkSpecs   `toml:"trusted_proxies"`
	TrustedHeaders     []string       `toml:"trusted_headers"`
	AllowedOrigins     []string       `toml:"allowed_origins"`
	AccessLog          string         `toml:"access_log"`
	AccessLogSizeLimit int64          `toml:"access_log_size_limit"`
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
		httpServerOptions = append(httpServerOptions, httpserver.WithTrustedProxyPolicy(httpserver.AllowNetworks("trusted proxies", c.TrustedProxies.IPNets())))
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

type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
	Process bool   `toml:"process"`
}

//go:embed config_defaults.toml
var configDefaultsData string

func DefaultConfig() (*Config, error) {
	config := &Config{}
	meta, err := toml.Decode(configDefaultsData, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config defaults (cause: %w)", err)
	}
	for _, key := range meta.Undecoded() {
		slog.Warn("unexpected default configuration key", slog.Any("key", key))
	}
	return config, nil
}

func LoadConfig(path string, strict bool) (*Config, error) {
	logArgPath := slog.String("path", path)
	slog.Info("loading config", logArgPath)
	config, err := DefaultConfig()
	if err != nil {
		return nil, err
	}
	meta, err := toml.DecodeFile(path, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config '%s' (cause: %w)", path, err)
	}
	strictViolation := false
	for _, key := range meta.Undecoded() {
		strictViolation = true
		slog.Warn("unexpected configuration key", logArgPath, slog.Any("key", key))
	}
	if strict && strictViolation {
		return nil, fmt.Errorf("config contains unexpected keys")
	}
	return config, nil
}

type LogLevel slog.Level

var knownLogLevels map[string]LogLevel = map[string]LogLevel{
	"debug": LogLevel(slog.LevelDebug),
	"info":  LogLevel(slog.LevelInfo),
	"warn":  LogLevel(slog.LevelWarn),
	"error": LogLevel(slog.LevelError),
}

func (l *LogLevel) Value() string {
	for value, level := range knownLogLevels {
		if *l == level {
			return value
		}
	}
	slog.Warn("unexpected log level", slog.Any("l", *l))
	return ""
}

func (l *LogLevel) MarshalTOML() ([]byte, error) {
	return []byte(`"` + l.Value() + `"`), nil
}

func (l *LogLevel) UnmarshalTOML(value any) error {
	levelString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected log level type %v", value)
	}
	level, ok := knownLogLevels[levelString]
	if !ok {
		return fmt.Errorf("unknown log level: '%s'", levelString)
	}
	*l = level
	return nil
}

type LogTarget log.Target

var knownLogTargets map[string]LogTarget = map[string]LogTarget{
	string(log.TargetStdout):     LogTarget(log.TargetStdout),
	string(log.TargetStdoutText): LogTarget(log.TargetStdoutText),
	string(log.TargetStdoutJSON): LogTarget(log.TargetStdoutJSON),
	string(log.TargetStderr):     LogTarget(log.TargetStderr),
	string(log.TargetStderrText): LogTarget(log.TargetStderrText),
	string(log.TargetStderrJSON): LogTarget(log.TargetStderrJSON),
	string(log.TargetFileText):   LogTarget(log.TargetFileText),
	string(log.TargetFileJSON):   LogTarget(log.TargetFileJSON),
	string(log.TargetSyslog):     LogTarget(log.TargetSyslog),
}

func (t *LogTarget) Value() string {
	for value, target := range knownLogTargets {
		if *t == target {
			return value
		}
	}
	slog.Warn("unexpected log target", slog.Any("t", *t))
	return ""
}

func (t *LogTarget) MarshalTOML() ([]byte, error) {
	return []byte(`"` + t.Value() + `"`), nil
}

func (t *LogTarget) UnmarshalTOML(value any) error {
	targetString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected log target type %v", value)
	}
	target, ok := knownLogTargets[targetString]
	if !ok {
		return fmt.Errorf("unknown log target: '%s'", targetString)
	}
	*t = target
	return nil
}

type LogColor log.Color

var knownLogColors map[string]LogColor = map[string]LogColor{
	"auto": LogColor(log.ColorAuto),
	"off":  LogColor(log.ColorOff),
	"on":   LogColor(log.ColorOn),
}

func (c *LogColor) Value() string {
	for value, color := range knownLogColors {
		if *c == color {
			return value
		}
	}
	slog.Warn("unexpected log color", slog.Any("c", *c))
	return ""
}

func (c *LogColor) MarshalTOML() ([]byte, error) {
	return []byte(`"` + c.Value() + `"`), nil
}

func (c *LogColor) UnmarshalTOML(value any) error {
	colorString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected log color type %v", value)
	}
	color, ok := knownLogColors[colorString]
	if !ok {
		return fmt.Errorf("unknown log color: '%s'", colorString)
	}
	*c = color
	return nil
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

type URLSpec struct {
	*url.URL
}

func (spec *URLSpec) Value() string {
	if spec.URL == nil {
		return ""
	}
	return spec.String()
}

func (spec *URLSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *URLSpec) UnmarshalTOML(value any) error {
	urlString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected URL type %v", value)
	}
	if urlString == "" {
		return nil
	}
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL: '%s' (cause: %w)", urlString, err)
	}
	spec.URL = parsedURL
	return nil
}

type URLSpecs []URLSpec

func (specs URLSpecs) URLs() []*url.URL {
	urls := make([]*url.URL, 0, len(specs))
	for _, spec := range specs {
		urls = append(urls, spec.URL)
	}
	return urls
}

type NetworkSpec struct {
	net.IPNet
}

func (spec *NetworkSpec) Value() string {
	return spec.String()
}

func (spec *NetworkSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.String() + `"`), nil
}

func (spec *NetworkSpec) UnmarshalTOML(value any) error {
	networkString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected network type %v", value)
	}
	_, parsedNetwork, err := net.ParseCIDR(networkString)
	if err != nil {
		return fmt.Errorf("invalid network: '%s' (cause: %w)", networkString, err)
	}
	spec.IPNet = *parsedNetwork
	return nil
}

type NetworkSpecs []NetworkSpec

func (specs NetworkSpecs) IPNets() []*net.IPNet {
	ipNets := make([]*net.IPNet, 0, len(specs))
	for _, spec := range specs {
		ipNets = append(ipNets, &spec.IPNet)
	}
	return ipNets
}
