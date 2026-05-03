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

package model

import "github.com/tdrn-org/netscanner/internal/i18n"

type EventDevice struct {
	ID              string
	Address         string
	Generation      int
	HardwareAddress string
	Network         string
	DNS             string
	Lat             float64
	Lng             float64
	City            i18n.Name
	Country         i18n.Name
	CountryCode     string
}
