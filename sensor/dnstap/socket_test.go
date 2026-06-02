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

package dnstap_test

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/sensor/dnstap"
)

func TestListenSocket(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dnstap.sock")

	receiver, err := dnstap.NewSocketReceiver(path, 0666, dnstap.DefaultMaxFrameSize, time.Unix(0, 0))
	require.NoError(t, err)

	go func() {
		receiver.Consume(func(entry *dnstap.Entry) {
			fmt.Println(entry.Decode())
		})
	}()

	log, err := os.Open("testdata/dnstap.log")
	require.NoError(t, err)
	defer log.Close()

	socket, err := net.Dial("unix", path)
	require.NoError(t, err)
	defer socket.Close()

	_, err = io.Copy(socket, log)
	require.NoError(t, err)
	time.Sleep(10 * time.Second)

	require.NoError(t, receiver.Shutdown(t.Context()))
	require.NoError(t, receiver.Close())
}
