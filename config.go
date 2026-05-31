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
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/tdrn-org/netscanner/config"
	"github.com/tdrn-org/netscanner/sensor/accesslog"
	"github.com/tdrn-org/netscanner/sensor/logfile"
	"github.com/tdrn-org/netscanner/sensor/syslog"
)

type Config struct {
	Logging   LoggingConfig   `toml:"logging"`
	Server    ServerConfig    `toml:"server"`
	Datastore DatastoreConfig `toml:"datastore"`
	Metrics   MetricsConfig   `toml:"metrics"`
	Sync      SyncConfig      `toml:"sync"`
	Sensors   SensorsConfig   `toml:"sensors"`
	ARPCache  ARPCacheConfig  `toml:"arp_cache"`
	DNS       DNSConfig       `toml:"dns"`
	GeoIP     GeoIPConfig     `toml:"geoip"`
}

type SensorsConfig struct {
	Include     string `toml:"include"`
	DefaultHost string `toml:"default_host"`
	sensors     []*SensorConfig
}

func (c *SensorsConfig) load(dir string, strict bool) error {
	include := c.Include
	if !filepath.IsAbs(c.Include) {
		include = filepath.Join(dir, c.Include)
	}
	entries, err := os.ReadDir(include)
	if errors.Is(err, os.ErrNotExist) {
		slog.Warn("no sensors include directory", slog.String("include", include))
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read sensors include directory '%s' (cause: %w)", include, err)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		sensorConfigFile := filepath.Join(include, entry.Name())
		logger := slog.With(slog.String("file", sensorConfigFile))
		sensorConfig := &SensorConfig{}
		logger.Info("loading sensor config")
		meta, err := toml.DecodeFile(sensorConfigFile, sensorConfig)
		if err != nil {
			return fmt.Errorf("failed to decode sensor config '%s' (cause: %w)", sensorConfigFile, err)
		}
		err = sensorConfig.validate()
		if err != nil {
			return fmt.Errorf("invalid sensor config '%s' (cause: %w)", sensorConfigFile, err)
		}
		strictViolation := false
		for _, key := range meta.Undecoded() {
			strictViolation = true
			logger.Warn("unexpected configuration key", slog.Any("key", key))
		}
		if strict && strictViolation {
			return fmt.Errorf("sensor config contains unexpected keys")
		}
		c.sensors = append(c.sensors, sensorConfig)
	}
	return nil
}

func (c *SensorsConfig) Configs() iter.Seq[*SensorConfig] {
	return slices.Values(c.sensors)
}

type SensorConfig struct {
	SyslogSensor    *SyslogSensorConfig    `toml:"syslog_sensor"`
	LogfileSensor   *LogfileSensorConfig   `toml:"logfile_sensor"`
	AccesslogSensor *AccesslogSensorConfig `toml:"accesslog_sensor"`
	DnstapSensor    *DnstapSensorConfig    `toml:"dnstap_sensor"`
}

type DnstapSensorConfig struct {
	Name string `toml:"name"`
	File string `toml:"file"`
}

func (c *SensorConfig) validate() error {
	sensorCount := 0
	if c.SyslogSensor != nil {
		sensorCount++
	}
	if c.LogfileSensor != nil {
		sensorCount++
	}
	if c.AccesslogSensor != nil {
		sensorCount++
	}
	if c.DnstapSensor != nil {
		sensorCount++
	}
	switch sensorCount {
	case 0:
		return fmt.Errorf("no sensor configuration")
	case 1:
		return nil
	default:
		return fmt.Errorf("non-unique sensor configuration")
	}
}

func (c *SensorConfig) String() string {
	if c.SyslogSensor != nil {
		return c.SyslogSensor.String()
	}
	if c.LogfileSensor != nil {
		return c.LogfileSensor.String()
	}
	if c.AccesslogSensor != nil {
		return c.AccesslogSensor.String()
	}
	if c.DnstapSensor != nil {
		return fmt.Sprintf(sensorStringFormatPath, "dnstap", c.DnstapSensor.Name, c.DnstapSensor.File)
	}
	return ""
}

const sensorStringFormatPath string = "%s/%s[%s]"
const sensorStringFormatURL string = "%s/%s[%s://%s]"

type SyslogSensorConfig struct {
	Name            string        `toml:"name"`
	Network         SyslogNetwork `toml:"network"`
	Address         string        `toml:"address"`
	LogMatcherIndex string        `toml:"log_matcher_index"`
}

func (c *SyslogSensorConfig) String() string {
	return fmt.Sprintf(sensorStringFormatURL, syslog.Name, c.Name, c.Network, c.Address)
}

type LogfileSensorConfig struct {
	Name            string `toml:"name"`
	Path            string `toml:"path"`
	LogMatcherIndex string `toml:"log_matcher_index"`
	Regexp          struct {
		Pattern         config.RegexpSpec `toml:"pattern"`
		TimestampField  int               `toml:"timestamp_field"`
		TimestampLayout string            `toml:"timestamp_layout"`
		HostField       int               `toml:"host_field"`
		MessageField    int               `toml:"message_field"`
	} `toml:"regexp"`
}

func (c *LogfileSensorConfig) String() string {
	return fmt.Sprintf(sensorStringFormatPath, logfile.Name, c.Name, c.Path)
}

func (c *LogfileSensorConfig) regexpScanOptions() *logfile.RegexpScanOptions {
	return &logfile.RegexpScanOptions{
		Pattern:         c.Regexp.Pattern.Regexp,
		TimestampField:  c.Regexp.TimestampField,
		TimestampLayout: c.Regexp.TimestampLayout,
		HostField:       c.Regexp.HostField,
		MessageField:    c.Regexp.MessageField,
		Tail:            true, // for now we default to true (accept misses; before duplicates)
	}
}

type AccesslogSensorConfig struct {
	Name       string      `toml:"name"`
	Path       string      `toml:"path"`
	AuthURIs   []string    `toml:"auth_uris"`
	IgnoreURIs []string    `toml:"ignore_uris"`
	Encoding   LogEncoding `toml:"encoding"`
	Regexp     struct {
		Pattern         config.RegexpSpec `toml:"pattern"`
		TimestampField  int               `toml:"timestamp_field"`
		TimestampLayout string            `toml:"timestamp_layout"`
		StatusField     int               `toml:"status_field"`
		AddressField    int               `toml:"address_field"`
		UserField       int               `toml:"user_field"`
		URIField        int               `toml:"uri_field"`
	} `toml:"regexp"`
	JSON struct {
		TimestampField  []string `toml:"timestamp_field"`
		TimestampLayout string   `toml:"timestamp_layout"`
		StatusField     []string `toml:"status_field"`
		AddressField    []string `toml:"address_field"`
		UserField       []string `toml:"user_field"`
		URIField        []string `toml:"uri_field"`
	} `toml:"json"`
}

func (c *AccesslogSensorConfig) String() string {
	return fmt.Sprintf(sensorStringFormatPath, logfile.Name, c.Name, c.Path)
}

func (c *AccesslogSensorConfig) regexpScanOptions() *accesslog.RegexpScanOptions {
	return &accesslog.RegexpScanOptions{
		ScanOptions: accesslog.ScanOptions{
			AuthURIs:   c.AuthURIs,
			IgnoreURIs: c.IgnoreURIs,
			Tail:       true, // for now we default to true (accept misses; before duplicates)
		},
		Pattern:         c.Regexp.Pattern.Regexp,
		TimestampField:  c.Regexp.TimestampField,
		TimestampLayout: c.Regexp.TimestampLayout,
		StatusField:     c.Regexp.StatusField,
		AddressField:    c.Regexp.AddressField,
		UserField:       c.Regexp.UserField,
	}
}

func (c *AccesslogSensorConfig) jsonScanOptions() *accesslog.JSONScanOptions {
	return &accesslog.JSONScanOptions{
		ScanOptions: accesslog.ScanOptions{
			AuthURIs:   c.AuthURIs,
			IgnoreURIs: c.IgnoreURIs,
			Tail:       true, // for now we default to true (accept misses; before duplicates)
		},
		TimestampField:  c.JSON.TimestampField,
		TimestampLayout: c.JSON.TimestampLayout,
		StatusField:     c.JSON.StatusField,
		AddressField:    c.JSON.AddressField,
		UserField:       c.JSON.UserField,
	}
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
	config.Sensors.sensors = []*SensorConfig{}
	return config, nil
}

func LoadConfig(file string, strict bool) (*Config, error) {
	config, err := loadRootConfig(file, strict)
	if err != nil {
		return nil, err
	}
	err = config.Sensors.load(filepath.Dir(file), strict)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func loadRootConfig(file string, strict bool) (*Config, error) {
	logger := slog.With(slog.String("file", file))
	logger.Info("loading config")
	config, err := DefaultConfig()
	if err != nil {
		return nil, err
	}
	meta, err := toml.DecodeFile(file, config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config '%s' (cause: %w)", file, err)
	}
	strictViolation := false
	for _, key := range meta.Undecoded() {
		strictViolation = true
		logger.Warn("unexpected configuration key", slog.Any("key", key))
	}
	if strict && strictViolation {
		return nil, fmt.Errorf("config contains unexpected keys")
	}
	return config, nil
}

type SyslogNetwork string

var knownSyslogNetworks map[string]SyslogNetwork = map[string]SyslogNetwork{
	"tcp":      SyslogNetwork("tcp"),
	"tcp4":     SyslogNetwork("tcp4"),
	"tcp6":     SyslogNetwork("tcp6"),
	"tcp+tls":  SyslogNetwork("tcp+tls"),
	"tcp4+tls": SyslogNetwork("tcp4+tls"),
	"tcp6+tls": SyslogNetwork("tcp6+tls"),
	"udp":      SyslogNetwork("udp"),
	"udp4":     SyslogNetwork("udp4"),
	"udp6":     SyslogNetwork("udp6"),
}

func (n *SyslogNetwork) Value() string {
	for value, network := range knownSyslogNetworks {
		if *n == network {
			return value
		}
	}
	slog.Warn("unexpected syslog network", slog.Any("n", *n))
	return ""
}

func (n *SyslogNetwork) MarshalTOML() ([]byte, error) {
	return []byte(`"` + n.Value() + `"`), nil
}

func (n *SyslogNetwork) UnmarshalTOML(value any) error {
	networkString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected syslog network type %v", value)
	}
	network, ok := knownSyslogNetworks[networkString]
	if !ok {
		return fmt.Errorf("unknown syslog network: '%s'", networkString)
	}
	*n = network
	return nil
}

type LogEncoding string

const (
	LogEncodingRegexp LogEncoding = "regexp"
	LogEncodingJSON   LogEncoding = "json"
)

var knownLogEncodings map[string]LogEncoding = map[string]LogEncoding{
	string(LogEncodingRegexp): LogEncodingRegexp,
	string(LogEncodingJSON):   LogEncodingJSON,
}

func (e *LogEncoding) Value() string {
	for value, encoding := range knownLogEncodings {
		if *e == encoding {
			return value
		}
	}
	slog.Warn("unexpected Log encoding", slog.Any("e", *e))
	return ""
}

func (e *LogEncoding) MarshalTOML() ([]byte, error) {
	return []byte(`"` + e.Value() + `"`), nil
}

func (e *LogEncoding) UnmarshalTOML(value any) error {
	logEncodingString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected Log encoding type %v", value)
	}
	logEncoding, ok := knownLogEncodings[logEncodingString]
	if !ok {
		return fmt.Errorf("unknown Log encoding: '%s'", logEncodingString)
	}
	*e = logEncoding
	return nil
}
