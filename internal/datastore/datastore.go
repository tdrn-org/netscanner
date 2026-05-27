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

package datastore

import (
	"context"
	"fmt"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/internal/device"
	"github.com/tdrn-org/netscanner/sensor"
)

type Store struct {
	driver *database.Driver
}

func New(driver *database.Driver) *Store {
	return &Store{
		driver: driver,
	}
}

func (s *Store) Close() error {
	return s.driver.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.driver.Ping(ctx)
}

func (s *Store) SelectLogMatcherIndexNames(ctx context.Context) ([]string, error) {
	return model.SelectLogMatcherIndexNames(ctx, s.driver)
}

func (s *Store) SelectOrInsertLogMatcherIndex(ctx context.Context, name string) (*model.LogMatcherIndex, error) {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	index, err := model.SelectLogMatcherIndexByName(txCtx, s.driver, name)
	if database.NoRows(err) {
		index = model.NewLogMatcherIndex(s.driver, name)
		err = index.Insert(txCtx)
	}
	if err != nil {
		return nil, err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return index, nil
}

func (s *Store) SelectDeviceByID(ctx context.Context, id string) (*model.Device, error) {
	return model.SelectDeviceByID(ctx, s.driver, id)
}

func (s *Store) SelectConnectionsByCursor(ctx context.Context) ([]*model.Connection, error) {
	return model.SelectConnectionsByCursor(ctx, s.driver)
}

func (s *Store) UpdateOrInsertConnection(ctx context.Context, serverInfo *device.Info, clientInfo *device.Info, event *sensor.Event) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	server, err := s.updateOrInsertDevice(txCtx, serverInfo)
	if err != nil {
		return err
	}
	client, err := s.updateOrInsertDevice(txCtx, clientInfo)
	if err != nil {
		return err
	}
	connection, err := model.SelectConnectionByServiceStatusUser(txCtx, s.driver, server, client, event.Service, model.ConnectionStatusFromSensorEventType(event.Type), event.User)
	if database.NoRows(err) {
		connection = model.NewConnection(s.driver, server, client, event)
		err = connection.Insert(txCtx)
	} else if err == nil {
		connection.Count++
		eventTimestamp := database.Time2DB(event.Timestamp)
		if connection.Last < eventTimestamp {
			connection.Last = eventTimestamp
		}
		err = connection.Update(txCtx)
	}
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) updateOrInsertDevice(ctx context.Context, deviceInfo *device.Info) (*model.Device, error) {
	device, err := model.SelectDeviceByAddress(ctx, s.driver, deviceInfo.Address)
	fmt.Println("device     :", device)
	fmt.Println("device-info:", deviceInfo)
	if err == nil {
		if !device.EqualDeviceInfo(deviceInfo) {
			device = model.NewDevice(s.driver, deviceInfo, device.Generation+1)
			err = device.Insert(ctx)
			fmt.Println("new-device :", device)
		}
	} else if database.NoRows(err) {
		device = model.NewDevice(s.driver, deviceInfo, 0)
		err = device.Insert(ctx)
	}
	return device, err
}
