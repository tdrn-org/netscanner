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

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
)

type LogMatcherIndex struct {
	driver  *database.Driver
	Name    string
	Version int
	Entries []*LogMatcherIndexEntry
}

func NewLogMatcherIndex(driver *database.Driver, name string) *LogMatcherIndex {
	return &LogMatcherIndex{
		driver:  driver,
		Name:    name,
		Version: 0,
		Entries: []*LogMatcherIndexEntry{},
	}
}

//go:embed log_matcher_index.select_by_name.sql
var logMatcherIndexSelectByNameSQL string

func SelectLogMatcherIndexByName(ctx context.Context, driver *database.Driver, name string) (*LogMatcherIndex, error) {
	txCtx, tx, err := driver.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, logMatcherIndexSelectByNameSQL, name)
	if err != nil {
		return nil, err
	}
	i := &LogMatcherIndex{
		driver: driver,
		Name:   name,
	}
	err = row.Scan(&i.Version)
	if database.NoRows(err) {
		commitErr := tx.CommitTx(txCtx)
		if commitErr != nil {
			err = commitErr
		}
	}
	if err != nil {
		return nil, err
	}
	entries, err := selectLogMatcherIndexEntriesByName(txCtx, tx, name)
	if err != nil {
		return nil, err
	}
	i.Entries = entries

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (i *LogMatcherIndex) ToIndex() *logmatcher.Index {
	index := logmatcher.NewIndex(i.Name)
	for _, entry := range i.Entries {
		index.AddMatch(entry.Service, entry.EventType, logmatcher.ParseMatch(entry.Match)...)
	}
	return index
}

//go:embed log_matcher_index.insert.sql
var logMatcherIndexInsertSQL string

func (i *LogMatcherIndex) Insert(ctx context.Context) error {
	txCtx, tx, err := i.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	err = tx.ExecTx(txCtx, logMatcherIndexInsertSQL, i.Name, i.Version)
	if err != nil {
		return err
	}
	for _, entry := range i.Entries {
		err = entry.insert(txCtx, tx, i.Name)
		if err != nil {
			return err
		}
	}

	return tx.CommitTx(txCtx)
}

//go:embed log_matcher_index.update_by_name.sql
var logMatcherIndexUpdateByNameSQL string

func (i *LogMatcherIndex) Update(ctx context.Context) error {
	txCtx, tx, err := i.driver.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.RollbackUncommitedTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, logMatcherIndexSelectByNameSQL, i.Name)
	if err != nil {
		return err
	}
	var datastoreVersion int
	err = row.Scan(&datastoreVersion)
	if err != nil {
		return err
	}
	if i.Version != datastoreVersion {
		return sql.ErrNoRows
	}
	i.Version++
	err = tx.ExecTx(txCtx, logMatcherIndexUpdateByNameSQL, i.Version, i.Name)
	if err != nil {
		return err
	}
	err = deleteLogMatcherIndexEntriesByName(txCtx, tx, i.Name)
	if err != nil {
		return err
	}
	for _, entry := range i.Entries {
		err = entry.insert(txCtx, tx, i.Name)
		if err != nil {
			return err
		}
	}

	return tx.CommitTx(txCtx)
}

type LogMatcherIndexEntry struct {
	Service   string
	EventType sensor.EventType
	Match     string
}

func NewLogMatcherIndexEntry(service string, eventType sensor.EventType, match string) *LogMatcherIndexEntry {
	return &LogMatcherIndexEntry{
		Service:   service,
		EventType: eventType,
		Match:     match,
	}
}

//go:embed log_matcher_index.select_entries_by_name.sql
var logMatcherIndexSelectEntriesByNameSQL string

func selectLogMatcherIndexEntriesByName(ctx context.Context, tx *database.Tx, name string) ([]*LogMatcherIndexEntry, error) {
	rows, err := tx.QueryTx(ctx, logMatcherIndexSelectEntriesByNameSQL, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []*LogMatcherIndexEntry{}
	for rows.Next() {
		entry := &LogMatcherIndexEntry{}
		err = rows.Scan(&entry.Service, &entry.EventType, &entry.Match)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

//go:embed log_matcher_index.delete_entries_by_name.sql
var logMatcherIndexDeleteEntriesByNameSQL string

func deleteLogMatcherIndexEntriesByName(ctx context.Context, tx *database.Tx, name string) error {
	return tx.ExecTx(ctx, logMatcherIndexDeleteEntriesByNameSQL, name)
}

//go:embed log_matcher_index.insert_entry.sql
var logMatcherIndexInsertEntrySQL string

func (lme *LogMatcherIndexEntry) insert(ctx context.Context, tx *database.Tx, name string) error {
	return tx.ExecTx(ctx, logMatcherIndexInsertEntrySQL, name, lme.Service, lme.EventType, lme.Match)
}
