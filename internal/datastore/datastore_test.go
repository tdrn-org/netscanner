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

package datastore_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-database/memory"
	"github.com/tdrn-org/netscanner/internal/datastore"
	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/sensor"
)

func TestLogMatcherIndex(t *testing.T) {
	store := datastore.New(newDatastore(t))

	// Insert
	index1, err := store.SelectOrCreateLogMatcherIndex(t.Context(), "test")
	require.NoError(t, err)
	require.Equal(t, "test", index1.Name)
	require.Equal(t, 0, index1.Version)
	require.Len(t, index1.Entries, 0)

	// Update
	index1.Entries = append(index1.Entries, model.NewLogMatcherIndexEntry("service", sensor.EventTypeGranted, "<match>"))
	err = index1.Update(t.Context())
	require.NoError(t, err)

	// Select
	index2, err := store.SelectOrCreateLogMatcherIndex(t.Context(), index1.Name)
	require.NoError(t, err)
	require.Equal(t, index1.Name, index2.Name)
	require.Equal(t, 1, index2.Version)
	require.Len(t, index2.Entries, 1)
}

func newDatastore(t *testing.T) *database.Driver {
	datastore, err := database.Open(memory.NewConfig(model.SqliteSchemaScriptOption))
	require.NoError(t, err)
	from, to, err := datastore.UpdateSchema(t.Context())
	require.NoError(t, err)
	require.Equal(t, database.SchemaNone, from)
	require.Equal(t, 1, to)
	return datastore
}
