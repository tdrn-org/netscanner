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

package logfile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/tdrn-org/netscanner/internal/file"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
)

const Name string = "logfile"

type Sensor interface {
	Path() string
	sensor.EventSource
}

func ScanRegexp(index *logmatcher.Index, path string, options *RegexpScanOptions) (Sensor, error) {
	err := options.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid regexp scan opations (cause: %w)", err)
	}
	sensor := &regexpLogfileSensor{
		index:   index,
		options: *options,
		scanner: file.NewScanner(path, &file.RegexpDecoder{Pattern: options.Pattern}, options.Tail),
		logger:  slog.With(slog.String("name", Name), slog.String("path", path)),
	}
	return sensor, nil
}

type RegexpScanOptions struct {
	Pattern         *regexp.Regexp
	TimestampField  int
	TimestampLayout string
	HostField       int
	MessageField    int
	Tail            bool
}

func (o *RegexpScanOptions) validate() error {
	if o.Pattern == nil {
		return fmt.Errorf("missing pattern")
	}
	maxFields := o.Pattern.NumSubexp()
	if o.Pattern == nil || maxFields < 4 {
		return fmt.Errorf("invalid pattern")
	}
	validationErrs := make([]error, 0, 5)
	if o.TimestampField < 1 || maxFields < o.TimestampField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid timestamp field option: %d", o.TimestampField))
	}
	if o.TimestampLayout == "" {
		validationErrs = append(validationErrs, fmt.Errorf("invalid timestamp layout option: '%s'", o.TimestampLayout))
	}
	if o.HostField < 1 || maxFields < o.HostField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid host field option: %d", o.HostField))
	}
	if o.MessageField < 1 || maxFields < o.MessageField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid message field option: %d", o.MessageField))
	}
	return errors.Join(validationErrs...)
}

func (o *RegexpScanOptions) resolve(index *logmatcher.Index, match []string) (*sensor.Event, error) {
	maxField := o.MessageField
	if o.HostField > maxField {
		maxField = o.HostField
	}
	if o.TimestampField > maxField {
		maxField = o.TimestampField
	}
	if len(match) <= maxField {
		return nil, nil
	}
	timestampMatch := match[o.TimestampField]
	timestamp, err := time.Parse(o.TimestampLayout, timestampMatch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse logfile timestamp '%s' (cause: %w)", timestampMatch, err)
	}
	host := match[o.HostField]
	message := match[o.MessageField]
	tokenizer := logmatcher.FieldsTokenizer
	tokens := tokenizer.Tokens(message)
	resolved := index.ResolveValues(tokens)
	if resolved == nil {
		return nil, nil
	}
	event := sensor.NewEvent()
	event.Host = host
	event.Timestamp = timestamp
	event.Type = resolved.EventType
	event.Address = resolved.Address
	event.HardwareAddress = resolved.HardwareAddress
	event.User = resolved.User
	event.Service = resolved.Service
	return event, nil
}

type regexpLogfileSensor struct {
	index     *logmatcher.Index
	options   RegexpScanOptions
	scanner   *file.Scanner[[]string]
	stopFunc  context.CancelFunc
	stoppedWG sync.WaitGroup
	logger    *slog.Logger
}

func (s *regexpLogfileSensor) Path() string {
	return s.scanner.Path()
}

func (s *regexpLogfileSensor) Name() string {
	return Name
}

func (s *regexpLogfileSensor) Collect(receiver sensor.EventReceiver) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	s.stopFunc = stop
	s.stoppedWG.Add(1)
	defer s.stoppedWG.Done()

	s.logger.Info("scanning...")
	for ctx.Err() == nil {
		_, match, err := s.scanner.Read()
		if err != nil {
			s.logger.Warn("scan failure", slog.Any("err", err))
			continue
		}
		event, err := s.options.resolve(s.index, match)
		event.Release()
		if err != nil {
			s.logger.Warn("resolve failure", slog.Any("err", err))
			continue
		}
		if event == nil {
			continue
		}
		event.Sensor = Name
		receiver.Queue(ctx, event)
	}
	s.logger.Info("stopped")
	return nil
}

func (s *regexpLogfileSensor) Shutdown(_ context.Context) error {
	stopFunc := s.stopFunc
	if stopFunc != nil {
		stopFunc()
		s.stoppedWG.Wait()
	}
	return nil
}

func (s *regexpLogfileSensor) Close() error {
	return s.scanner.Close()
}
