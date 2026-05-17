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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"

	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/internal/i18n"
	"github.com/tdrn-org/netscanner/ouidb"
)

var ErrDeviceNotFound error = errors.New("device not found")

func (s *Server) GetDevice(ctx context.Context, id string) (*DeviceInfo, error) {
	device, err := s.store.SelectDeviceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info (cause: %w)", err)
	}
	if device == nil {
		return nil, ErrDeviceNotFound
	}
	deviceInfo := s.deviceToDeviceInfo(ctx, device)
	return deviceInfo, nil
}

func (s *Server) deviceToDeviceInfo(ctx context.Context, device *model.Device) *DeviceInfo {
	locale := i18n.Locale(ctx)
	hardwareVendor := ""
	if device.HardwareAddress != "" {
		hardwareAddress, err := net.ParseMAC(device.HardwareAddress)
		if err == nil {
			vendor, err := ouidb.DefaultIndexReader().LookupHardwareAddr(hardwareAddress)
			if err == nil {
				hardwareVendor = vendor.Name
			} else {
				s.logger.Error("failed to lookup vendor", slog.String("mac", hardwareAddress.String()), slog.Any("err", err))
			}
		} else {
			s.logger.Error("invalid MAC", slog.String("mac", device.HardwareAddress), slog.Any("err", err))
		}
	}
	lat := math.NaN()
	if device.Lat.Valid {
		lat = device.Lat.Float64
	}
	lng := math.NaN()
	if device.Lng.Valid {
		lat = device.Lng.Float64
	}
	return &DeviceInfo{
		ID:              device.ID,
		Address:         device.Address,
		Network:         device.Network,
		HardwareAddress: device.HardwareAddress,
		HardwareVendor:  hardwareVendor,
		DNS:             device.DNS,
		Lat:             lat,
		Lng:             lng,
		City:            device.City[locale],
		Country:         device.Country[locale],
		CountryCode:     device.CountryCode,
	}
}
