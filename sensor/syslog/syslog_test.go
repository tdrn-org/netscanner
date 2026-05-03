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

package syslog_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/go-tlsconf"
	"github.com/tdrn-org/go-tlsconf/tlsclient"
	"github.com/tdrn-org/go-tlsconf/tlsserver"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/syslog"
)

const syslogListenAddress string = "localhost:0"

func TestSyslogSensorTCP(t *testing.T) {
	sensor, err := syslog.ListenTCP(testIndex(t), "tcp", syslogListenAddress)
	require.NoError(t, err)
	defer sensor.Close()
	runSyslogSensor(t, sensor, "tcp")
}

func TestSyslogSensorTLS(t *testing.T) {
	sensor, err := syslog.ListenTLS(testIndex(t), "tcp", syslogListenAddress, tlsserver.GetConfig())
	require.NoError(t, err)
	defer sensor.Close()
	runSyslogSensor(t, sensor, "tcp+tls")
}

func TestSyslogSensorUDP(t *testing.T) {
	sensor, err := syslog.ListenUDP(testIndex(t), "udp", syslogListenAddress)
	require.NoError(t, err)
	defer sensor.Close()
	runSyslogSensor(t, sensor, "udp")
}

func runSyslogSensor(t *testing.T, sensor syslog.Sensor, network string) {
	logsFile, err := os.Open("testdata/logs.txt")
	require.NoError(t, err)
	defer logsFile.Close()
	logs := bufio.NewReader(logsFile)
	go func() {
		err := sensor.Collect(syslogReceiver)
		require.NoError(t, err)
	}()
	config := log.Config{
		Level:         slog.LevelDebug.String(),
		Target:        log.TargetSyslog,
		SyslogAddress: sensor.Address(),
		SyslogNetwork: network,
	}
	logger, _ := config.GetLogger(nil)
	for {
		log, err := logs.ReadString('\n')
		eof := errors.Is(err, io.EOF)
		if err != nil && !eof {
			require.NoError(t, err)
		}
		log = strings.TrimSpace(log)
		logger.Warn(log)
		if eof {
			break
		}
	}
	time.Sleep(time.Second)
	err = sensor.Shutdown(t.Context())
	require.NoError(t, err)
}

var syslogReceiver sensor.EventReceiverFunc = func(_ context.Context, event *sensor.Event) {
	fmt.Println(event.String())
}

func testIndex(t *testing.T) *logmatcher.Index {
	index := logmatcher.NewIndex("syslog")
	file, err := os.Open("testdata/index.txt")
	require.NoError(t, err)
	defer file.Close()
	err = index.Load(file)
	require.NoError(t, err)
	return index
}

func init() {
	tlsserver.SetOptions(tlsserver.UseEphemeralCertificate("localhost", tlsconf.CertificateAlgorithmDefault, time.Hour))
	tlsclient.SetOptions(tlsconf.EnableInsecureSkipVerify())
}
