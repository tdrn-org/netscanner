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
	_ "embed"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"

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
	Lat             float64
	Lng             float64
	City            i18n.Name
	Country         i18n.Name
	CountryCode     string
}

func NewDevice(driver *database.Driver, deviceInfo *device.Info, generation int) *Device {
	lat := deviceInfo.Geo.Lat
	if math.IsNaN(lat) {
		lat = 0.0
	}
	lng := deviceInfo.Geo.Lng
	if math.IsNaN(lng) {
		lng = 0.0
	}
	return &Device{
		driver:          driver,
		ID:              database.NewID(),
		Address:         deviceInfo.Address.String(),
		Generation:      generation,
		Network:         deviceInfo.Network,
		DNS:             deviceInfo.DNS,
		HardwareAddress: deviceInfo.HardwareAddress.String(),
		Lat:             lat,
		Lng:             lng,
		City:            deviceInfo.Geo.City,
		Country:         deviceInfo.Geo.Country,
		CountryCode:     "",
	}
}

func (d *Device) String() string {
	buffer := &strings.Builder{}
	buffer.WriteString("Address:")
	buffer.WriteString(d.Address)
	buffer.WriteString(" Network:")
	buffer.WriteString(d.Network)
	if d.HardwareAddress != "" {
		buffer.WriteString(" MAC:")
		buffer.WriteString(d.HardwareAddress)
	}
	if d.DNS != "" {
		buffer.WriteString(" DNS:")
		buffer.WriteString(d.DNS)
	}
	buffer.WriteString(" Loc:")
	buffer.WriteString(strconv.FormatFloat(d.Lat, 'f', 2, 64))
	buffer.WriteString(",")
	buffer.WriteString(strconv.FormatFloat(d.Lng, 'f', 2, 64))
	if len(d.City) > 0 {
		buffer.WriteString(" City:")
		buffer.WriteString(i18n.Name(d.City).String())
	}
	if len(d.Country) > 0 {
		buffer.WriteString(" Country:")
		buffer.WriteString(i18n.Name(d.Country).String())
	}
	return buffer.String()
}

func (d *Device) EqualDeviceInfo(deviceInfo *device.Info) bool {
	if d.Address != deviceInfo.Address.String() {
		fmt.Println("address-mismatch")
		return false
	}
	if d.Network != deviceInfo.Network {
		fmt.Println("network-mismatch")
		return false
	}
	if d.DNS != deviceInfo.DNS {
		fmt.Println("dns-mismatch")
		return false
	}
	if d.HardwareAddress != deviceInfo.HardwareAddress.String() {
		fmt.Println("hardware-address-mismatch")
		return false
	}
	if math.Abs(d.Lat-deviceInfo.Geo.Lat) > 0.0001 {
		fmt.Println("lat-mismatch")
		return false
	}
	if math.Abs(d.Lng-deviceInfo.Geo.Lng) > 0.0001 {
		fmt.Println("lng-mismatch")
		return false
	}
	if !d.City.Equal(deviceInfo.Geo.City) {
		fmt.Println("city-mismatch")
		return false
	}
	if !d.Country.Equal(deviceInfo.Geo.Country) {
		fmt.Println("country-mismatch")
		return false
	}
	if d.CountryCode != deviceInfo.Geo.CountryCode {
		fmt.Println("country-code-mismatch")
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
	err = row.Scan(&d.Address, &d.Generation, &d.Network, &d.DNS, &d.HardwareAddress, &d.Lat, &d.Lng, &d.City, &d.Country, &d.CountryCode)
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

	addressString := address.String()
	row, err := tx.QueryRowTx(txCtx, deviceSelectByAddressSQL, addressString)
	if err != nil {
		return nil, err
	}
	d := &Device{
		driver:  driver,
		Address: addressString,
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
