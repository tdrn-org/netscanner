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
	"sync"

	"github.com/tdrn-org/netscanner/internal/file"
	"github.com/tdrn-org/netscanner/sensor"
)

type JSONScanOptions struct {
	ScanOptions
	TimestampField  file.JSONPath
	TimestampLayout string
	StatusField     file.JSONPath
	AddressField    file.JSONPath
	UserField       file.JSONPath
	URIField        file.JSONPath
}

func (o *JSONScanOptions) validate() error {
	validationErrs := make([]error, 0, 3)
	if o.TimestampField.Len() == 0 {
		validationErrs = append(validationErrs, errors.New("missing timestamp field option"))
	}
	if o.StatusField.Len() == 0 {
		validationErrs = append(validationErrs, errors.New("missing status field option"))
	}
	if o.AddressField.Len() == 0 {
		validationErrs = append(validationErrs, errors.New("missing address field option"))
	}
	return errors.Join(validationErrs...)
}

func (o *JSONScanOptions) resolve(object file.JSON) (*sensor.Event, error) {
	timestamp, err := file.JSONValueToTime(o.TimestampLayout, object, o.TimestampField...)
	if err != nil {
		return nil, err
	}
	status, err := file.JSONValueToInt(object, o.StatusField...)
	if err != nil {
		return nil, err
	}
	addressString, err := file.JSONValue[string](object, o.AddressField...)
	if err != nil {
		return nil, err
	}
	address, err := netip.ParseAddr(addressString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address field '%s' (cause: %w)", addressString, err)
	}
	user, _ := file.JSONValue[string](object, o.UserField...)
	uri, _ := file.JSONValue[string](object, o.URIField...)
	if o.isIgnoreURI(uri) {
		return nil, nil
	}
	eventType := sensor.EventTypeInformational
	switch {
	case 200 <= status && status < 300:
		if o.isAuthURI(uri) {
			eventType = sensor.EventTypeGranted
		}
	case 400 <= status && status < 500:
		eventType = sensor.EventTypeDenied
	case 500 <= status && status < 600:
		eventType = sensor.EventTypeError
	}
	event := &sensor.Event{
		Timestamp: timestamp,
		Type:      eventType,
		Address:   address,
		User:      user,
	}
	return event, nil
}

type jsonAccesslogSensor struct {
	options   JSONScanOptions
	scanner   *file.Scanner[file.JSON]
	stopFunc  context.CancelFunc
	stoppedWG sync.WaitGroup
	logger    *slog.Logger
}

func (s *jsonAccesslogSensor) Path() string {
	return s.scanner.Path()
}

func (s *jsonAccesslogSensor) Name() string {
	return Name
}

func (s *jsonAccesslogSensor) Collect(receiver sensor.EventReceiver) error {
	ctx, stop := context.WithCancel(context.Background())
	s.stopFunc = stop
	s.stoppedWG.Go(func() {
		s.logger.Info("scanning...")
		for ctx.Err() == nil {
			_, object, err := s.scanner.Read()
			if err != nil {
				s.logger.Warn("scan failure", slog.Any("err", err))
				continue
			}
			if object == nil {
				continue
			}
			event, err := s.options.resolve(object)
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

func (s *jsonAccesslogSensor) Shutdown(_ context.Context) error {
	stopFunc := s.stopFunc
	if stopFunc != nil {
		stopFunc()
		s.stoppedWG.Wait()
	}
	return nil
}

func (s *jsonAccesslogSensor) Close() error {
	return s.scanner.Close()
}
