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

import (
	"context"
	"database/sql"
	_ "embed"
	"math"
	"net/netip"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/internal/device"
	"github.com/tdrn-org/netscanner/internal/i18n"
)

type Device struct {
	driver          *database.Driver
	ID              string
	Address         string
	Generation      int
	Network         string
	DNS             string
	HardwareAddress string
	Lat             sql.NullFloat64
	Lng             sql.NullFloat64
	City            i18n.Name
	Country         i18n.Name
	CountryCode     string
}

func NewDevice(driver *database.Driver, deviceInfo *device.Info, generation int) *Device {
	return &Device{
		driver:          driver,
		ID:              database.NewID(),
		Address:         deviceInfo.Address.StringExpanded(),
		Generation:      generation,
		Network:         deviceInfo.Network,
		DNS:             deviceInfo.DNS,
		HardwareAddress: deviceInfo.HardwareAddress.String(),
		Lat: sql.NullFloat64{
			Float64: deviceInfo.Geo.Lat,
			Valid:   !math.IsNaN(deviceInfo.Geo.Lat),
		},
		Lng: sql.NullFloat64{
			Float64: deviceInfo.Geo.Lng,
			Valid:   !math.IsNaN(deviceInfo.Geo.Lng),
		},
		City:        deviceInfo.Geo.City,
		Country:     deviceInfo.Geo.Country,
		CountryCode: "",
	}
}

func (d *Device) EqualDeviceInfo(deviceInfo *device.Info) bool {
	if d.Address != deviceInfo.Address.StringExpanded() {
		return false
	}
	if d.Network != deviceInfo.Network {
		return false
	}
	if d.DNS != deviceInfo.DNS {
		return false
	}
	if d.HardwareAddress != deviceInfo.HardwareAddress.String() {
		return false
	}
	if math.IsNaN(deviceInfo.Geo.Lat) {
		if d.Lat.Valid {
			return false
		}
	} else if !d.Lat.Valid {
		return false
	} else if d.Lat.Float64 != deviceInfo.Geo.Lat {
		return false
	}
	if math.IsNaN(deviceInfo.Geo.Lng) {
		if d.Lng.Valid {
			return false
		}
	} else if !d.Lng.Valid {
		return false
	} else if d.Lng.Float64 != deviceInfo.Geo.Lng {
		return false
	}
	if !d.City.Equal(deviceInfo.Geo.City) {
		return false
	}
	if !d.Country.Equal(deviceInfo.Geo.Country) {
		return false
	}
	if d.CountryCode != deviceInfo.Geo.CountryCode {
		return false
	}
	return true
}

//go:embed device.select_by_id.sql
var deviceSelectByIDSQL string

func SelectDeviceByID(ctx context.Context, driver *database.Driver, id string) (*Device, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, deviceSelectByIDSQL, id)
	if err != nil {
		return nil, err
	}
	d := &Device{
		driver: driver,
		ID:     id,
	}
	err = row.Scan(&d.Generation, &d.Address, &d.Network, &d.DNS, &d.HardwareAddress, &d.Lat, &d.Lng, &d.City, &d.Country, &d.CountryCode)
	if database.NoRows(err) {
		commitErr := tx.CommitTx(txCtx)
		if commitErr != nil {
			err = commitErr
		}
	}
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return d, nil
}

//go:embed device.select_by_address.sql
var deviceSelectByAddressSQL string

func SelectDeviceByAddress(ctx context.Context, driver *database.Driver, address netip.Addr) (*Device, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	addressExpanded := address.StringExpanded()
	row, err := tx.QueryRowTx(txCtx, deviceSelectByAddressSQL, addressExpanded)
	if err != nil {
		return nil, err
	}
	d := &Device{
		driver:  driver,
		Address: addressExpanded,
	}
	err = row.Scan(&d.ID, &d.Generation, &d.Network, &d.DNS, &d.HardwareAddress, &d.Lat, &d.Lng, &d.City, &d.Country, &d.CountryCode)
	if database.NoRows(err) {
		commitErr := tx.CommitTx(txCtx)
		if commitErr != nil {
			err = commitErr
		}
	}
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return d, nil
}

//go:embed device.insert.sql
var deviceInsertSQL string

func (d *Device) Insert(ctx context.Context) error {
	txCtx, tx, err := d.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, deviceInsertSQL, d.ID, d.Generation, d.Address, d.Network, d.DNS, d.HardwareAddress, d.Lat, d.Lng, d.City, d.Country, d.CountryCode)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
