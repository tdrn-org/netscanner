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
	"errors"
	"fmt"
	"os"
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
	collectDone := make(chan error, 1)
	go func() {
		collectDone <- accesslogSensor.Collect(sensor.EventReceiverFunc(func(ctx context.Context, event *sensor.Event) {
			fmt.Println(event)
		}))
	}()
	// Wait briefly for the collector to drain the testdata file and reach EOF.
	// After EOF the scanner idles (with a 250ms backoff per read), so we shut
	// down explicitly via Shutdown/Close rather than a blind time.Sleep.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-collectDone:
			require.NoError(t, err)
			require.NoError(t, accesslogSensor.Close())
			return
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Still collecting — explicitly shut it down.
	require.NoError(t, accesslogSensor.Shutdown(t.Context()))
	select {
	case err := <-collectDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("collector did not finish after Shutdown")
	}
	// Close is idempotent on a closed sensor; ErrClosed from double-close is OK.
	if err := accesslogSensor.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		t.Fatalf("unexpected close error: %v", err)
	}
}
