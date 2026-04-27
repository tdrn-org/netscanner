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

func (s *Store) SelectOrCreateLogMatcherIndex(ctx context.Context, name string) (*model.LogMatcherIndex, error) {
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
