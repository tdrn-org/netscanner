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
	"net/netip"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/internal/device"
	"github.com/tdrn-org/netscanner/internal/i18n"
)

type EventDevice struct {
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

func NewEventDevice(driver *database.Driver, deviceInfo *device.Info, generation int) *EventDevice {
	return &EventDevice{
		driver:          driver,
		ID:              database.NewID(),
		Address:         deviceInfo.Address.StringExpanded(),
		Generation:      generation,
		Network:         deviceInfo.Network,
		DNS:             deviceInfo.DNS.Name,
		HardwareAddress: "",
		Lat:             sql.NullFloat64{},
		Lng:             sql.NullFloat64{},
		City:            i18n.Name{},
		Country:         i18n.Name{},
		CountryCode:     "",
	}
}

func (d *EventDevice) EqualDeviceInfo(deviceInfo *device.Info) bool {
	return d.Address == deviceInfo.Address.StringExpanded() && d.Network == deviceInfo.Network && d.DNS == deviceInfo.DNS.Name
}

//go:embed event_device.select_by_address.sql
var eventDeviceSelectByAddressSQL string

func SelectEventDeviceByAddress(ctx context.Context, driver *database.Driver, address netip.Addr) (*EventDevice, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	addressExpanded := address.StringExpanded()
	row, err := tx.QueryRowTx(txCtx, eventDeviceSelectByAddressSQL, addressExpanded)
	if err != nil {
		return nil, err
	}
	d := &EventDevice{
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

//go:embed event_device.insert.sql
var eventDeviceInsertSQL string

func (d *EventDevice) Insert(ctx context.Context) error {
	txCtx, tx, err := d.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, eventDeviceInsertSQL, d.ID, d.Address, d.Generation, d.Network, d.DNS, d.HardwareAddress, d.Lat, d.Lng, d.City, d.Country, d.CountryCode)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
