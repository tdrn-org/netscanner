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

package accesslog

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/tdrn-org/netscanner/internal/file"
	"github.com/tdrn-org/netscanner/sensor"
)

const Name string = "accesslog"

type Sensor interface {
	Path() string
	sensor.EventSource
}

type ScanOptions struct {
	AuthURIs   []string
	IgnoreURIs []string
	Tail       bool
}

func (o *ScanOptions) isAuthURI(uri string) bool {
	for _, authURI := range o.AuthURIs {
		if strings.HasPrefix(uri, authURI) {
			return true
		}
	}
	return false
}

func (o *ScanOptions) isIgnoreURI(uri string) bool {
	for _, ignoreURI := range o.IgnoreURIs {
		if strings.HasPrefix(uri, ignoreURI) {
			return true
		}
	}
	return false
}

func (o *ScanOptions) mapHttpStatus(status int, uri string) sensor.EventType {
	eventType := sensor.EventTypeInformational
	switch {
	case 200 <= status && status < 300:
		if o.isAuthURI(uri) {
			eventType = sensor.EventTypeGranted
		}
	case status == 401 || status == 404:
		// considere these informational, as they occuring also during "normal" access patterns
		break
	case 400 <= status && status < 500:
		eventType = sensor.EventTypeDenied
	case 500 <= status && status < 600:
		eventType = sensor.EventTypeError
	}
	return eventType
}

func ScanRegexp(path string, options *RegexpScanOptions) (Sensor, error) {
	err := options.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid regexp scan opations (cause: %w)", err)
	}
	sensor := &regexpAccesslogSensor{
		options: *options,
		scanner: file.NewScanner(path, &file.RegexpDecoder{Pattern: options.Pattern}, options.Tail),
		logger:  slog.With(slog.String("name", Name), slog.String("path", path), slog.String("encoding", "regexp")),
	}
	return sensor, nil
}

func ScanJSON(path string, options *JSONScanOptions) (Sensor, error) {
	err := options.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid json scan opations (cause: %w)", err)
	}
	sensor := &jsonAccesslogSensor{
		options: *options,
		scanner: file.NewScanner(path, &file.JSONDecoder{}, options.Tail),
		logger:  slog.With(slog.String("name", Name), slog.String("path", path), slog.String("encoding", "json")),
	}
	return sensor, nil
}
