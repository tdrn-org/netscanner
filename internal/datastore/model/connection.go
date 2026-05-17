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

type ConnectionStatus string

const (
	ConnectionStatusInformational ConnectionStatus = "informational"
	ConnectionStatusGranted       ConnectionStatus = "granted"
	ConnectionStatusDenied        ConnectionStatus = "denied"
	ConnectionStatusError         ConnectionStatus = "error"
)

func ConnectionStatusFromSensorEventType(eventType sensor.EventType) ConnectionStatus {
	switch eventType {
	case sensor.EventTypeInformational:
		return ConnectionStatusInformational
	case sensor.EventTypeGranted:
		return ConnectionStatusGranted
	case sensor.EventTypeDenied:
		return ConnectionStatusDenied
	default:
		return ConnectionStatusError
	}
}

type Connection struct {
	driver *database.Driver
	ID     string
	Server *Device
	Client *Device
	Status ConnectionStatus
	User   string
	Count  int64
	First  int64
	Last   int64
}

func NewConnection(driver *database.Driver, server *Device, client *Device, event *sensor.Event) *Connection {
	now := database.Now()
	return &Connection{
		driver: driver,
		ID:     database.NewID(),
		Server: server,
		Client: client,
		Status: ConnectionStatusFromSensorEventType(event.Type),
		User:   event.User,
		Count:  1,
		First:  now,
		Last:   now,
	}
}

//go:embed connection.select_by_status_user.sql
var connectionSelectByStatusUserSQL string

func SelectConnectionByStatusUser(ctx context.Context, driver *database.Driver, server *Device, client *Device, status ConnectionStatus, user string) (*Connection, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, connectionSelectByStatusUserSQL, server.ID, client.ID, status, user)
	if err != nil {
		return nil, err
	}
	a := &Connection{
		driver: driver,
		Server: server,
		Client: client,
		Status: status,
		User:   user,
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

//go:embed connection.insert.sql
var connectionInsertSQL string

func (a *Connection) Insert(ctx context.Context) error {
	txCtx, tx, err := a.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, connectionInsertSQL, a.ID, a.Server.ID, a.Client.ID, a.Status, a.User, a.Count, a.First, a.Last)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}

//go:embed connection.update.sql
var connectionUpdateSQL string

func (a *Connection) Update(ctx context.Context) error {
	txCtx, tx, err := a.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, connectionUpdateSQL, a.Count, a.Last, a.ID)
	if err != nil {
		return err
	}

	return tx.CommitTx(txCtx)
}
