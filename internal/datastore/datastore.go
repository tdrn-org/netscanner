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

func (s *Store) UpdateOrInsertEvent(ctx context.Context, event *sensor.Event, deviceInfo *device.Info) error {
	txCtx, tx, err := s.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	target, err := model.SelectEventTargetByHostService(txCtx, s.driver, event.Host, event.Service)
	if database.NoRows(err) {
		target = model.NewEventTarget(s.driver, event.Host, event.Service)
		err = target.Insert(txCtx)
	}
	if err != nil {
		return err
	}

	device, err := model.SelectEventDeviceByAddress(txCtx, s.driver, event.Address)
	if err == nil {
		if !device.EqualDeviceInfo(deviceInfo) {
			device = model.NewEventDevice(s.driver, deviceInfo, device.Generation+1)
			err = device.Insert(txCtx)
		}
	} else if database.NoRows(err) {
		device = model.NewEventDevice(s.driver, deviceInfo, 0)
		err = device.Insert(txCtx)
	}
	if err != nil {
		return err
	}

	action, err := model.SelectEventActionByUserStatus(txCtx, s.driver, target, device, event.User, model.EventActionStatusFromEventType(event.Type))
	if database.NoRows(err) {
		action = model.NewEventAction(s.driver, target, device, event)
		err = action.Insert(txCtx)
	} else if err == nil {
		action.Count++
		eventTimestamp := database.Time2DB(event.Timestamp)
		if action.Last < eventTimestamp {
			action.Last = eventTimestamp
		}
		err = action.Update(txCtx)
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
