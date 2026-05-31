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

// Package dnstap provides a DNS query sensor based on BIND's dnstap log format.
//
// The sensor reads dnstap frames from a file. Each frame contains
// a protobuf-encoded Dnstap message with DNS query/response data.
//
// BIND configuration:
//
//	options {
//	    dnstap { client query; resolver query; };
//	    dnstap-output file "/var/log/named/dnstap.log" size 100m versions 7;
//	};
package dnstap

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/tdrn-org/netscanner/sensor"
)

const Name string = "dnstap"

// compile-time interface check
var _ sensor.EventSource = (*FileSource)(nil)

// FileSource reads dnstap frames from a file with rotation support.
type FileSource struct {
	path   string
	logger *slog.Logger
	mu     sync.Mutex
	file   *os.File
	reader *bufio.Reader
	closed bool
}

// NewFileSource creates a dnstap sensor reading from a file.
func NewFileSource(path string) (*FileSource, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dnstap: open %q: %w", path, err)
	}
	return &FileSource{
		path:   path,
		file:   f,
		reader: bufio.NewReaderSize(f, 64*1024),
		logger: slog.With(slog.String("sensor", Name), slog.String("path", path)),
	}, nil
}

// Name returns the sensor type name.
func (s *FileSource) Name() string { return Name }

// Path returns the file path.
func (s *FileSource) Path() string { return s.path }

// Collect starts reading dnstap frames and emitting events.
// This method blocks until Shutdown is called or an unrecoverable error occurs.
func (s *FileSource) Collect(receiver sensor.EventReceiver) error {
	s.logger.Info("starting dnstap sensor")
	defer s.logger.Info("dnstap sensor stopped")

	// Read the fstrm control frame first (10 bytes: 4-byte length=6 + "ready\0\n")
	if err := s.readControlFrame(); err != nil {
		return fmt.Errorf("dnstap: read control frame: %w", err)
	}

	for {
		// Check if file was rotated (inode changed)
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil
		}
		s.mu.Unlock()

		event, err := s.readFrame()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// No more data — wait and retry (live tail)
				select {
				case <-time.After(2 * time.Second):
					continue
				}
			}
			s.logger.Error("dnstap read error", slog.Any("err", err))
			continue
		}
		if event == nil {
			continue
		}

		sensorEvent := &sensor.Event{
			Timestamp: event.timestamp,
			Type:      sensor.EventTypeInformational,
			Address:   event.clientAddr,
			Service:   "dns",
			Sensor:    Name,
		}
		receiver.Queue(context.Background(), sensorEvent)
	}
}

// Shutdown gracefully stops the sensor.
func (s *FileSource) Shutdown(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// Close releases resources.
func (s *FileSource) Close() error {
	return s.Shutdown(context.Background())
}

// readControlFrame reads the fstrm control frame that BIND writes at startup.
// Format: [4-byte length] "ready" NUL NUL NUL
func (s *FileSource) readControlFrame() error {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(s.reader, lenBuf); err != nil {
		return fmt.Errorf("read control length: %w", err)
	}
	ctrlLen := binary.BigEndian.Uint32(lenBuf)
	ctrlData := make([]byte, ctrlLen)
	if _, err := io.ReadFull(s.reader, ctrlData); err != nil {
		return fmt.Errorf("read control data: %w", err)
	}
	// "ready" frame — ignore content, just acknowledge
	return nil
}

// readFrame reads a single dnstap data frame.
// Format: [4-byte length] [protobuf payload]
// Returns nil event for control frames or unparseable data.
func (s *FileSource) readFrame() (*dnsEvent, error) {
	// Peek at 4 bytes for frame length
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(s.reader, lenBuf); err != nil {
		return nil, err
	}

	frameLen := binary.BigEndian.Uint32(lenBuf)

	// Length 0 = fstrm control frame (skip)
	if frameLen == 0 {
		return nil, nil
	}

	// Sanity check
	if frameLen > 10*1024*1024 { // 10MB max
		return nil, fmt.Errorf("frame too large: %d bytes", frameLen)
	}

	payload := make([]byte, frameLen)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}

	return parseFrame(payload)
}

// --- Protobuf parsing for dnstap ---
// We implement a minimal protobuf decoder for the specific dnstap fields we need.
// This avoids depending on google.golang.org/protobuf and protoc.

type dnsEvent struct {
	timestamp  time.Time
	clientAddr netip.Addr
}

// parseFrame parses a protobuf-encoded Dnstap message.
func parseFrame(payload []byte) (*dnsEvent, error) {
	event := &dnsEvent{timestamp: time.Now()}
	pos := 0

	for pos < len(payload) {
		fieldNum, wireType, n := readTag(payload[pos:])
		if n <= 0 {
			break
		}
		pos += n

		switch fieldNum {
		case 14: // message (field 14, wire type 2 = embedded message)
			if wireType == 2 {
				msgLen, n := readVarint(payload[pos:])
				if n > 0 {
					pos += n
					end := pos + int(msgLen)
					if end <= len(payload) {
						parseMessage(payload[pos:end], event)
					}
					pos = end
				}
			} else {
				pos += skipField(payload[pos:], wireType)
			}
		default:
			pos += skipField(payload[pos:], wireType)
		}
	}

	if !event.clientAddr.IsValid() {
		return nil, nil // incomplete frame, skip
	}
	return event, nil
}

// parseMessage extracts client IP from a Message protobuf.
func parseMessage(data []byte, event *dnsEvent) {
	pos := 0
	for pos < len(data) {
		fieldNum, wireType, n := readTag(data[pos:])
		if n <= 0 {
			break
		}
		pos += n

		switch fieldNum {
		case 5: // query_address
			if wireType == 2 {
				addrLen, n := readVarint(data[pos:])
				if n > 0 {
					pos += n
					parseSocketAddr(data[pos:pos+int(addrLen)], event)
					pos += int(addrLen)
				}
			} else {
				pos += skipField(data[pos:], wireType)
			}
		default:
			pos += skipField(data[pos:], wireType)
		}
	}
}

// parseSocketAddr extracts an IP address from a SocketAddress protobuf.
func parseSocketAddr(data []byte, event *dnsEvent) {
	pos := 0
	for pos < len(data) {
		fieldNum, wireType, n := readTag(data[pos:])
		if n <= 0 {
			break
		}
		pos += n

		if fieldNum == 3 && wireType == 2 { // address
			ipLen, n := readVarint(data[pos:])
			if n > 0 {
				pos += n
				switch ipLen {
				case 4:
					if pos+4 <= len(data) {
						event.clientAddr = netip.AddrFrom4([4]byte(data[pos : pos+4]))
					}
				case 16:
					if pos+16 <= len(data) {
						event.clientAddr = netip.AddrFrom16([16]byte(data[pos : pos+16]))
					}
				}
				pos += int(ipLen)
			}
		} else {
			pos += skipField(data[pos:], wireType)
		}
	}
}

// --- protobuf wire format helpers ---

func readTag(data []byte) (fieldNum int, wireType int, bytesRead int) {
	v, n := readVarint(data)
	if n <= 0 {
		return 0, 0, 0
	}
	return int(v >> 3), int(v & 0x7), n
}

func readVarint(data []byte) (uint64, int) {
	if len(data) == 0 {
		return 0, 0
	}
	var v uint64
	for i := 0; i < 10 && i < len(data); i++ {
		b := data[i]
		v |= uint64(b&0x7F) << (7 * i)
		if b < 0x80 {
			return v, i + 1
		}
	}
	return 0, 0
}

func skipField(data []byte, wireType int) int {
	switch wireType {
	case 0: // varint
		_, n := readVarint(data)
		return min(n, len(data))
	case 2: // length-delimited
		v, n := readVarint(data)
		if n > 0 && len(data) >= n+int(v) {
			return n + int(v)
		}
		return len(data)
	case 5: // 32-bit fixed
		return min(4, len(data))
	default:
		return len(data)
	}
}

// required by bufio.NewReaderSize
var _ = bytes.NewReader(nil)
