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
	"context"

	"github.com/tdrn-org/netscanner/logmatcher"
)

func (s *Server) ListLogMatcherIndexNames(ctx context.Context) ([]string, error) {
	return s.store.SelectLogMatcherIndexNames(ctx)
}

func (s *Server) resolveLogMatcherIndex(ctx context.Context, name string) (*logmatcher.Index, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.resolveLogMatcherLocked(ctx, name)
}

func (s *Server) resolveLogMatcherLocked(ctx context.Context, name string) (*logmatcher.Index, error) {
	index := s.logMatchers[name]
	if index != nil {
		return index, nil
	}
	indexModel, err := s.store.SelectOrInsertLogMatcherIndex(ctx, name)
	if err != nil {
		return nil, err
	}
	index = indexModel.ToIndex()
	s.logMatchers[name] = index
	return index, nil
}
