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

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/sensor"
)

type EventActionStatus string

const (
	EventActionStatusGranted EventActionStatus = "granted"
	EventActionStatusDenied  EventActionStatus = "denied"
	EventActionStatusError   EventActionStatus = "error"
)

func EventActionStatusFromEventType(eventType sensor.EventType) EventActionStatus {
	switch eventType {
	case sensor.EventTypeGranted:
		return EventActionStatusGranted
	case sensor.EventTypeDenied:
		return EventActionStatusDenied
	default:
		return EventActionStatusError
	}
}

type EventAction struct {
	driver *database.Driver
	ID     string
	Target *EventTarget
	Device *EventDevice
	User   string
	Status EventActionStatus
	Count  int64
	First  int64
	Last   int64
}

func NewEventAction(driver *database.Driver, target *EventTarget, device *EventDevice, event *sensor.Event) *EventAction {
	now := database.Now()
	return &EventAction{
		driver: driver,
		ID:     database.NewID(),
		Target: target,
		Device: device,
		User:   event.User,
		Status: EventActionStatusFromEventType(event.Type),
		Count:  1,
		First:  now,
		Last:   now,
	}
}

//go:embed event_action.select_by_user_status.sql
var eventActionSelectByUserStatusSQL string

func SelectEventActionByUserStatus(ctx context.Context, driver *database.Driver, target *EventTarget, device *EventDevice, user string, status EventActionStatus) (*EventAction, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, eventActionSelectByUserStatusSQL, target.ID, device.ID, user, status)
	if err != nil {
		return nil, err
	}
	a := &EventAction{
		driver: driver,
		Target: target,
		Device: device,
		User:   user,
		Status: status,
	}
	err = row.Scan(&a.ID, &a.Count, &a.First, &a.Last)
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
	return a, nil
}

//go:embed event_action.insert.sql
var eventActionInsertSQL string

func (a *EventAction) Insert(ctx context.Context) error {
	txCtx, tx, err := a.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, eventActionInsertSQL, a.ID, a.Target.ID, a.Device.ID, a.User, a.Status, a.Count, a.First, a.Last)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

//go:embed event_action.update.sql
var eventActionUpdateSQL string

func (a *EventAction) Update(ctx context.Context) error {
	txCtx, tx, err := a.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, eventActionUpdateSQL, a.Count, a.Last, a.ID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
