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

package accesslog_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/accesslog"
)

func TestAccesslogJSON(t *testing.T) {
	options := &accesslog.JSONScanOptions{
		TimestampField: []string{"ts"},
		StatusField:    []string{"status"},
		AddressField:   []string{"request", "remote_ip"},
	}
	accesslogSensor, err := accesslog.ScanJSON("testdata/access_log.json", options)
	require.NoError(t, err)
	err = accesslogSensor.Collect(sensor.EventReceiverFunc(func(ctx context.Context, event *sensor.Event) {
		fmt.Println(event)
	}))
	require.NoError(t, err)
	time.Sleep(5 * time.Second)
	err = accesslogSensor.Shutdown(t.Context())
	require.NoError(t, err)
	err = accesslogSensor.Close()
	require.NoError(t, err)
}
