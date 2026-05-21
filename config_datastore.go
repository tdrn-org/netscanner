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

package netscanner

import (
	"fmt"
	"log/slog"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-database/memory"
	"github.com/tdrn-org/go-database/postgres"
	"github.com/tdrn-org/go-database/sqlite"
	"github.com/tdrn-org/netscanner/internal/datastore/model"
)

type DatastoreConfig struct {
	DatabaseType DatabaseType `toml:"type"`
	MemoryConfig struct {     /* no parameters */
	} `toml:"memory"`
	SQLiteConfig struct {
		File string `toml:"file"`
	} `toml:"sqlite"`
	PostgresConfig struct {
		Address  string `toml:"address"`
		DBName   string `toml:"db"`
		User     string `toml:"user"`
		Password string `toml:"password"`
	} `toml:"postgres"`
}

func (c *DatastoreConfig) config() (database.Config, error) {
	switch c.DatabaseType {
	case DatabaseType(memory.Type):
		return memory.NewConfig(model.SqliteSchemaScriptOption), nil
	case DatabaseType(sqlite.Type):
		return sqlite.NewConfig(c.SQLiteConfig.File, sqlite.ModeRWC, model.SqliteSchemaScriptOption), nil
	case DatabaseType(postgres.Type):
		return postgres.NewConfig(c.PostgresConfig.DBName, c.PostgresConfig.User, c.PostgresConfig.Password, postgres.WithAddress(c.PostgresConfig.Address), model.PostgresSchemaScriptOption)
	}
	return nil, fmt.Errorf("unrecognized datastore type '%s'", c.DatabaseType)
}

type DatabaseType database.Type

var knownDatabaseTypes map[string]DatabaseType = map[string]DatabaseType{
	string(memory.Type):   DatabaseType(memory.Type),
	string(sqlite.Type):   DatabaseType(sqlite.Type),
	string(postgres.Type): DatabaseType(postgres.Type),
}

func (t *DatabaseType) Value() string {
	for value, databaseType := range knownDatabaseTypes {
		if *t == databaseType {
			return value
		}
	}
	slog.Warn("unexpected database type", slog.Any("t", *t))
	return ""
}

func (t *DatabaseType) MarshalTOML() ([]byte, error) {
	return []byte(`"` + t.Value() + `"`), nil
}

func (t *DatabaseType) UnmarshalTOML(value any) error {
	databaseTypeString, ok := value.(string)
	if !ok {
		return fmt.Errorf("unexpected database type type %v", value)
	}
	databaseType, ok := knownDatabaseTypes[databaseTypeString]
	if !ok {
		return fmt.Errorf("unknown database type: '%s'", databaseTypeString)
	}
	*t = databaseType
	return nil
}
