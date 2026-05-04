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
)

type EventTarget struct {
	driver  *database.Driver
	ID      string
	Host    string
	Service string
}

func NewEventTarget(driver *database.Driver, host string, service string) *EventTarget {
	return &EventTarget{
		driver:  driver,
		ID:      database.NewID(),
		Host:    host,
		Service: service,
	}
}

//go:embed event_target.select_by_host_service.sql
var eventTargetSelectByHostServiceSQL string

func SelectEventTargetByHostService(ctx context.Context, driver *database.Driver, host string, service string) (*EventTarget, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, eventTargetSelectByHostServiceSQL, host, service)
	if err != nil {
		return nil, err
	}
	t := &EventTarget{
		driver:  driver,
		Host:    host,
		Service: service,
	}
	err = row.Scan(&t.ID)
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
	return t, nil
}

//go:embed event_target.insert.sql
var eventTargetInsertSQL string

func (t *EventTarget) Insert(ctx context.Context) error {
	txCtx, tx, err := t.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, eventTargetInsertSQL, t.ID, t.Host, t.Service)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
