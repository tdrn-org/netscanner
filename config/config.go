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

package config

import (
	"fmt"
	"net/netip"
	"net/url"
	"regexp"
	"time"
)

type DurationSpec time.Duration

func (spec *DurationSpec) Value() string {
	return time.Duration(*spec).String()
}

func (spec *DurationSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *DurationSpec) UnmarshalTOML(value any) error {
	durationString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected duration type %v", value)
	}
	parsedDuration, err := time.ParseDuration(durationString)
	if err != nil {
		return fmt.Errorf("invalid duration: '%s' (cause: %w)", durationString, err)
	}
	*spec = DurationSpec(parsedDuration)
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
	netip.Prefix
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
	parsedNetwork, err := netip.ParsePrefix(networkString)
	if err != nil {
		return fmt.Errorf("invalid network: '%s' (cause: %w)", networkString, err)
	}
	spec.Prefix = parsedNetwork
	return nil
}

type NetworkSpecs []NetworkSpec

func (specs NetworkSpecs) Prefixes() []netip.Prefix {
	networks := make([]netip.Prefix, 0, len(specs))
	for _, spec := range specs {
		networks = append(networks, spec.Prefix)
	}
	return networks
}

type RegexpSpec struct {
	*regexp.Regexp
}

func (spec *RegexpSpec) Value() string {
	if spec.Regexp == nil {
		return ""
	}
	return spec.String()
}

func (spec *RegexpSpec) MarshalTOML() ([]byte, error) {
	return []byte(`"` + spec.Value() + `"`), nil
}

func (spec *RegexpSpec) UnmarshalTOML(value any) error {
	regexpString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected Regexp type %v", value)
	}
	if regexpString == "" {
		return nil
	}
	parsedRegexp, err := regexp.Compile(regexpString)
	if err != nil {
		return fmt.Errorf("invalid Regexp: '%s' (cause: %w)", regexpString, err)
	}
	spec.Regexp = parsedRegexp
	return nil
}
