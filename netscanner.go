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
	"fmt"
	"os"

	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/syslog"
)

func Run(ctx context.Context, args []string) error {
	syslogIndex := logmatcher.NewIndex("syslog", logmatcher.FieldsTokenizer)
	syslogIndexFile, err := os.Open("syslog_index.txt")
	if err != nil {
		return err
	}
	defer syslogIndexFile.Close()
	err = syslogIndex.Load(syslogIndexFile)
	if err != nil {
		return err
	}
	syslogSensor, err := syslog.ListenTCP(syslogIndex, "tcp", "localhost:9514")
	if err != nil {
		return err
	}
	defer syslogSensor.Close()
	err = syslogSensor.Collect(sensor.EventReceiverFunc(func(event *sensor.Event) {
		fmt.Println(event)
	}))
	return err
}
