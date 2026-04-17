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
	"errors"
	"fmt"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/netscanner/sensor"
)

type Profile struct {
	datastore     *database.Driver
	ID            string
	Name          string
	LogMatchers   map[string]*LogMatcher
	SyslogSensors map[string]*SyslogSensor
}

func NewProfile(datastore *database.Driver, name string) *Profile {
	return &Profile{
		datastore:     datastore,
		ID:            database.NewID(),
		Name:          name,
		LogMatchers:   make(map[string]*LogMatcher),
		SyslogSensors: make(map[string]*SyslogSensor),
	}
}

func DeleteProfileByName(ctx context.Context, datastore *database.Driver, name string) error {
	txCtx, tx, err := datastore.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.EndTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, profileSelectByNameSQL, name)
	if err != nil {
		return err
	}
	var id string
	err = row.Scan(&id)
	if err == nil {
		err = DeleteProfileByID(txCtx, datastore, id)
		if err != nil {
			return err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("select server profile failure (cause: %w)", err)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}
	return nil
}

//go:embed profile.delete_syslog_sensor_by_id.sql
var profileDeleteSyslogSensorByIDSQL string

//go:embed profile.delete_log_matcher_entry_by_id.sql
var profileDeleteLogMatcherEntryByIDSQL string

//go:embed profile.delete_log_matcher_by_id.sql
var profileDeleteLogMatcherByIDSQL string

//go:embed profile.delete_by_id.sql
var profileDeleteByIDSQL string

func DeleteProfileByID(ctx context.Context, datastore *database.Driver, id string) error {
	txCtx, tx, err := datastore.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.EndTx(txCtx)

	err = tx.ExecTx(txCtx, profileDeleteSyslogSensorByIDSQL, id)
	if err != nil {
		return fmt.Errorf("delete syslog sensor failure (cause: %w)", err)
	}
	err = tx.ExecTx(txCtx, profileDeleteLogMatcherEntryByIDSQL, id)
	if err != nil {
		return fmt.Errorf("delete log matcher entry failure (cause: %w)", err)
	}
	err = tx.ExecTx(txCtx, profileDeleteLogMatcherByIDSQL, id)
	if err != nil {
		return fmt.Errorf("delete log matcher failure (cause: %w)", err)
	}
	err = tx.ExecTx(txCtx, profileDeleteByIDSQL, id)
	if err != nil {
		return fmt.Errorf("delete profile failure (cause: %w)", err)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}
	return nil
}

//go:embed profile.select_by_name.sql
var profileSelectByNameSQL string

func SelectProfileByName(ctx context.Context, datastore *database.Driver, name string) (*Profile, error) {
	txCtx, tx, err := datastore.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.EndTx(txCtx)

	row, err := tx.QueryRowTx(txCtx, profileSelectByNameSQL, name)
	if err != nil {
		return nil, err
	}
	profile := &Profile{
		datastore:     datastore,
		Name:          name,
		LogMatchers:   make(map[string]*LogMatcher),
		SyslogSensors: make(map[string]*SyslogSensor),
	}
	err = row.Scan(&profile.ID)
	if errors.Is(err, sql.ErrNoRows) {
		profile = nil
	} else if err != nil {
		return nil, fmt.Errorf("select server profile failure (cause: %w)", err)
	} else {
		logMatchers, err := selectProfileLogMatchers(txCtx, tx, profile.ID)
		if err != nil {
			return nil, err
		}
		for _, logMatcher := range logMatchers {
			profile.LogMatchers[logMatcher.Name] = logMatcher
		}
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

//go:embed profile.select_log_matchers.sql
var profileSelectLogMatchersSQL string

func selectProfileLogMatchers(ctx context.Context, tx *database.Tx, profileID string) ([]*LogMatcher, error) {
	rows, err := tx.QueryTx(ctx, profileSelectLogMatchersSQL, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logMatchers := make([]*LogMatcher, 0)
	for rows.Next() {
		logMatcher := &LogMatcher{}
		err = rows.Scan(&logMatcher.ID, &logMatcher.Name, &logMatcher.Tokenizer)
		if err != nil {
			return nil, err
		}
		logMatcherEntries, err := selectProfileLogMatcherEntries(ctx, tx, logMatcher.ID)
		if err != nil {
			return nil, err
		}
		logMatcher.Entries = logMatcherEntries
		logMatchers = append(logMatchers, logMatcher)
	}
	return logMatchers, nil
}

//go:embed profile.select_log_matcher_entries.sql
var profileSelectLogMatcherEntriesSQL string

func selectProfileLogMatcherEntries(ctx context.Context, tx *database.Tx, logMatcherID string) ([]*LogMatcherEntry, error) {
	rows, err := tx.QueryTx(ctx, profileSelectLogMatcherEntriesSQL, logMatcherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logMatcherEntries := make([]*LogMatcherEntry, 0)
	for rows.Next() {
		logMatcherEntry := &LogMatcherEntry{}
		err = rows.Scan(&logMatcherEntry.ID, &logMatcherEntry.Service, &logMatcherEntry.EventType, &logMatcherEntry.Match)
		if err != nil {
			return nil, err
		}
		logMatcherEntries = append(logMatcherEntries, logMatcherEntry)
	}
	return logMatcherEntries, nil
}

//go:embed profile.select_all.sql
var profileSelectAllSQL string

func SelectProfileNames(ctx context.Context, datastore *database.Driver) ([]string, error) {
	txCtx, tx, err := datastore.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.EndTx(txCtx)

	rows, err := tx.QueryTx(txCtx, profileSelectAllSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make([]string, 0)
	for rows.Next() {
		var id string
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			return nil, fmt.Errorf("query profile names failure (cause: %w)", err)
		}
		names = append(names, name)
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return nil, err
	}
	return names, nil
}

//go:embed profile.insert.sql
var profileInsertSQL string

func (p *Profile) Insert(ctx context.Context) error {
	txCtx, tx, err := p.datastore.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.EndTx(txCtx)

	err = tx.ExecTx(txCtx, profileInsertSQL, p.ID, p.Name)
	if err != nil {
		return err
	}
	for _, logMatcher := range p.LogMatchers {
		err = logMatcher.insert(txCtx, tx, p.ID)
		if err != nil {
			return err
		}
	}
	for _, syslogSensor := range p.SyslogSensors {
		err = syslogSensor.insert(txCtx, tx, p.ID)
		if err != nil {
			return err
		}
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}
	return nil
}

type LogMatcher struct {
	ID        string
	Name      string
	Tokenizer string
	Entries   []*LogMatcherEntry
}

func NewLogMatcher(name string, tokenizer string) *LogMatcher {
	return &LogMatcher{
		ID:        database.NewID(),
		Name:      name,
		Tokenizer: tokenizer,
		Entries:   make([]*LogMatcherEntry, 0),
	}
}

//go:embed profile.insert_log_matcher.sql
var profileInsertLogMatcherSQL string

func (lm *LogMatcher) insert(ctx context.Context, tx *database.Tx, profileID string) error {
	err := tx.ExecTx(ctx, profileInsertLogMatcherSQL, lm.ID, profileID, lm.Name, lm.Tokenizer)
	if err != nil {
		return err
	}
	for _, entry := range lm.Entries {
		err = entry.insert(ctx, tx, lm.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

type LogMatcherEntry struct {
	ID        string
	Service   string
	EventType sensor.EventType
	Match     string
}

func NewLogMatcherEntry(service string, eventType sensor.EventType, match string) *LogMatcherEntry {
	return &LogMatcherEntry{
		ID:        database.NewID(),
		Service:   service,
		EventType: eventType,
		Match:     match,
	}
}

//go:embed profile.insert_log_matcher_entry.sql
var profileInsertLogMatcherEntrySQL string

func (lme *LogMatcherEntry) insert(ctx context.Context, tx *database.Tx, logMatcherID string) error {
	return tx.ExecTx(ctx, profileInsertLogMatcherEntrySQL, lme.ID, logMatcherID, lme.Service, lme.EventType, lme.Match)
}

type SyslogSensor struct {
	ID         string
	Name       string
	Enabled    bool
	Network    string
	Address    string
	LogMatcher string
}

func NewSyslogSensor(name string, enabled bool, network, address, logMatcher string) *SyslogSensor {
	return &SyslogSensor{
		ID:         database.NewID(),
		Name:       name,
		Enabled:    enabled,
		Network:    network,
		Address:    address,
		LogMatcher: logMatcher,
	}
}

//go:embed profile.insert_syslog_sensor.sql
var profileInsertSyslogSensorSQL string

func (s *SyslogSensor) insert(ctx context.Context, tx *database.Tx, profileID string) error {
	return tx.ExecTx(ctx, profileInsertSyslogSensorSQL, s.ID, profileID, s.Name, s.Enabled, s.Network, s.Address, s.LogMatcher)
}
