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

package accesslog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/tdrn-org/netscanner/internal/file"
	"github.com/tdrn-org/netscanner/sensor"
)

type RegexpScanOptions struct {
	ScanOptions
	Pattern         *regexp.Regexp
	TimestampField  int
	TimestampLayout string
	StatusField     int
	AddressField    int
	UserField       int
	URIField        int
}

func (o *RegexpScanOptions) validate() error {
	if o.Pattern == nil {
		return fmt.Errorf("missing pattern")
	}
	maxFields := o.Pattern.NumSubexp()
	if o.Pattern == nil || maxFields < 4 {
		return fmt.Errorf("invalid pattern")
	}
	validationErrs := make([]error, 0, 6)
	if o.TimestampField < 1 || maxFields < o.TimestampField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid timestamp field option: %d", o.TimestampField))
	}
	if o.TimestampLayout == "" {
		validationErrs = append(validationErrs, fmt.Errorf("invalid timestamp layout option: '%s'", o.TimestampLayout))
	}
	if o.StatusField < 1 || maxFields < o.StatusField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid status field option: %d", o.StatusField))
	}
	if o.AddressField < 1 || maxFields < o.AddressField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid address field option: %d", o.AddressField))
	}
	if o.UserField < 0 || maxFields < o.UserField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid user field option: %d", o.UserField))
	}
	if o.URIField < 0 || maxFields < o.URIField {
		validationErrs = append(validationErrs, fmt.Errorf("invalid URI field option: %d", o.URIField))
	}
	return errors.Join(validationErrs...)
}

func (o *RegexpScanOptions) resolve(match []string) (*sensor.Event, error) {
	timestampMatch := match[o.TimestampField]
	timestamp, err := time.Parse(o.TimestampLayout, timestampMatch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access_log timestamp '%s' (cause: %w)", timestampMatch, err)
	}
	statusMatch := match[o.StatusField]
	status, err := strconv.ParseInt(statusMatch, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access_log status code '%s' (cause: %w)", statusMatch, err)
	}
	if status < 100 || 599 < status {
		return nil, fmt.Errorf("invalid access_log status code %d", status)
	}
	addressMatch := match[o.AddressField]
	address, err := netip.ParseAddr(addressMatch)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access_log remote address code '%s' (cause: %w)", addressMatch, err)
	}
	user := ""
	if o.UserField > 0 {
		user = match[o.UserField]
	}
	uri := ""
	if o.URIField > 0 {
		uri = match[o.URIField]
	}
	if o.isIgnoreURI(uri) {
		return nil, nil
	}
	eventType := o.mapHttpStatus(int(status), uri)
	event := &sensor.Event{
		Timestamp: timestamp,
		Type:      eventType,
		Address:   address,
		User:      user,
	}
	return event, nil
}

type regexpAccesslogSensor struct {
	options   RegexpScanOptions
	scanner   *file.Scanner[[]string]
	stopFunc  context.CancelFunc
	stoppedWG sync.WaitGroup
	logger    *slog.Logger
}

func (s *regexpAccesslogSensor) Path() string {
	return s.scanner.Path()
}

func (s *regexpAccesslogSensor) Name() string {
	return Name
}

func (s *regexpAccesslogSensor) Collect(receiver sensor.EventReceiver) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	s.stopFunc = stop
	s.stoppedWG.Go(func() {
		s.logger.Info("scanning...")
		for ctx.Err() == nil {
			_, match, err := s.scanner.Read()
			if err != nil {
				s.logger.Warn("scan failure", slog.Any("err", err))
				continue
			}
			event, err := s.options.resolve(match)
			if err != nil {
				s.logger.Warn("resolve failure", slog.Any("err", err))
				continue
			}
			if event == nil {
				continue
			}
			event.Service = "http"
			event.Sensor = Name
			receiver.Queue(ctx, event)
		}
		s.logger.Info("stopping...")
	})
	return nil
}

func (s *regexpAccesslogSensor) Shutdown(_ context.Context) error {
	stopFunc := s.stopFunc
	if stopFunc != nil {
		stopFunc()
		s.stoppedWG.Wait()
	}
	return nil
}

func (s *regexpAccesslogSensor) Close() error {
	return s.scanner.Close()
}
