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

package geoip

import (
	"context"
	"fmt"
	"math"
	"net/netip"

	"github.com/mmcloughlin/geohash"
	"github.com/tdrn-org/netscanner/internal/i18n"
	"golang.org/x/text/language"
)

type ProviderName string

type ProviderConfig interface {
	ProviderName() ProviderName
}

type Info struct {
	Lat         float64
	Lng         float64
	City        map[language.Tag]string
	Country     map[language.Tag]string
	CountryCode string
}

var NoInfo *Info = &Info{
	Lat:     math.NaN(),
	Lng:     math.NaN(),
	City:    map[language.Tag]string{},
	Country: map[language.Tag]string{},
}

func (i *Info) Equal(i2 *Info) bool {
	return i.Lat == i2.Lat && i.Lng == i2.Lng && i18n.Name(i.City).Equal(i2.City) && i18n.Name(i.Country).Equal(i2.City) && i.CountryCode == i2.CountryCode
}

func (i *Info) IsNaN() bool {
	return math.IsNaN(i.Lat) || math.IsNaN(i.Lng)
}

func (i *Info) Hash(chars uint) string {
	return geohash.EncodeWithPrecision(i.Lat, i.Lng, chars)
}

type Provider interface {
	Name() ProviderName
	Lookup(ctx context.Context, address netip.Addr) (*Info, error)
	Close() error
}

type OpenProviderFunc func(config ProviderConfig) (Provider, error)

var providers map[ProviderName]OpenProviderFunc = make(map[ProviderName]OpenProviderFunc)

func RegisterProvider(name ProviderName, open OpenProviderFunc) {
	providers[name] = open
}

func Open(config ProviderConfig) (Provider, error) {
	name := config.ProviderName()
	open, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown GeoIP provider name '%s'", name)
	}
	return open(config)
}
