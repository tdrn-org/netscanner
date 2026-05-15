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

package file

import (
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Decoder[T any] interface {
	Nil() T
	Feed(r io.Reader) (int, error)
	Decode() (T, error)
}

type Scanner[T any] struct {
	path    string
	decoder Decoder[T]
	file    *os.File
	closed  bool
	tail    bool
	delay   time.Duration
	logger  *slog.Logger
	mutex   sync.Mutex
}

func NewScanner[T any](path string, decoder Decoder[T], tail bool) *Scanner[T] {
	return &Scanner[T]{
		path:    path,
		decoder: decoder,
		tail:    tail,
		logger:  slog.With(slog.String("scanner", path)),
	}
}

func (s *Scanner[T]) Path() string {
	return s.path
}

func (s *Scanner[T]) Read() (int, T, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return 0, s.decoder.Nil(), os.ErrClosed
	}
	if s.delay != 0 {
		time.Sleep(s.delay)
		s.delay = 0
	}
	if !s.ensureOpen() {
		return 0, s.decoder.Nil(), nil
	}
	read, err := s.decoder.Feed(s.file)
	if err == io.EOF {
		err = nil
		s.delay = defaultReadDelay
	} else if err != nil {
		return read, s.decoder.Nil(), err
	}
	decoded, err := s.decoder.Decode()
	return read, decoded, err
}

func (s *Scanner[T]) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return os.ErrClosed
	}
	s.closed = true
	if s.file == nil {
		return nil
	}
	return s.file.Close()
}

const defaultReadDelay time.Duration = 250 * time.Millisecond

func (s *Scanner[D]) ensureOpen() bool {
	if s.file == nil {
		s.logger.Debug("opening file...")
		file, err := os.Open(s.path)
		if err != nil {
			s.logger.Info("file not accessible; retrying after delay", slog.Any("cause", err))
			s.triggerReopen(2 * defaultReadDelay)
			return false
		}
		s.file = file
		if s.tail {
			_, err = file.Seek(0, 2)
			if err != nil {
				s.logger.Info("file seek failure; reopening file", slog.Any("cause", err))
				s.triggerReopen(0)
				return false
			}
			s.tail = false
		}
	}
	return true
}

func (s *Scanner[D]) triggerReopen(delay time.Duration) {
	s.close()
	s.delay = delay
}

func (s *Scanner[D]) close() {
	if s.file != nil {
		s.logger.Debug("closing file...")
		err := s.file.Close()
		if err != nil {
			s.logger.Warn("close file failure", slog.Any("err", err))
		}
		s.file = nil
	}
}

const minFeedSize int = 1024
const defaultFeedSize int = minFeedSize << 2

// TODO: Implement upper limit guard and recovery
// const bufferLimit int = minFeedSize << 6
