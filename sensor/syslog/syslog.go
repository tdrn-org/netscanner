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

package syslog

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/tdrn-org/go-log"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
)

const Name string = "syslog"

type Sensor interface {
	Address() string
	sensor.EventSource
}

func ListenTCP(index *logmatcher.Index, network, address string) (Sensor, error) {
	resolvedAddress, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve TCP address %s/%s (cause: %w)", network, address, err)
	}
	listener, err := net.ListenTCP(network, resolvedAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP listener on address %s (cause: %w)", resolvedAddress.String(), err)
	}
	return newTCPSensor(index, "tcp", listener), nil
}

func ListenTLS(index *logmatcher.Index, network, address string, config *tls.Config) (Sensor, error) {
	listener, err := tls.Listen(network, address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener on address %s (cause: %w)", address, err)
	}
	return newTCPSensor(index, "tls", listener), nil
}

func ListenUDP(index *logmatcher.Index, network, address string) (Sensor, error) {
	resolvedAddress, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address %s/%s (cause: %w)", network, address, err)
	}
	listener, err := net.ListenUDP(network, resolvedAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener on address %s (cause: %w)", resolvedAddress.String(), err)
	}
	return newUDPSensor(index, "udp", listener), nil
}

type tcpSensor struct {
	index    *logmatcher.Index
	listener net.Listener
	logger   *slog.Logger
}

func newTCPSensor(index *logmatcher.Index, proto string, listener net.Listener) Sensor {
	sensor := &tcpSensor{
		index:    index,
		listener: listener,
		logger:   slog.With(slog.String("sensor", Name), slog.String("proto", proto), slog.String("address", listener.Addr().String())),
	}
	sensor.logger.Info("listening")
	return sensor
}

func (s *tcpSensor) Address() string {
	return s.listener.Addr().String()
}

func (s *tcpSensor) Collect(receiver sensor.EventReceiver) error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				return err
			}
			return nil
		}
		go func() {
			s.collectHandler(receiver, conn)
		}()
	}
}

func (s *tcpSensor) collectHandler(receiver sensor.EventReceiver, conn net.Conn) {
	connLogger := s.logger.With(slog.String("remote", conn.RemoteAddr().String()))
	connLogger.Debug("new connection")
	defer conn.Close()
	for {
		decoder := &log.SyslogDecoder{}
		err := decoder.Read(conn)
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			connLogger.Debug("connection closed")
			return
		}
		if err != nil {
			connLogger.Error("read/decode failure", slog.Any("err", err))
			return
		}
		queueSyslogMessages(s.index, receiver, decoder, s.logger)
	}
}

func (s *tcpSensor) Shutdown(_ context.Context) error {
	return nil
}

func (s *tcpSensor) Close() error {
	return s.listener.Close()
}

type udpSensor struct {
	index    *logmatcher.Index
	listener *net.UDPConn
	logger   *slog.Logger
}

func newUDPSensor(index *logmatcher.Index, proto string, listener *net.UDPConn) Sensor {
	sensor := &udpSensor{
		index:    index,
		listener: listener,
		logger:   slog.With(slog.String("sensor", "syslog"), slog.String("proto", proto), slog.String("address", listener.LocalAddr().String())),
	}
	sensor.logger.Info("listening")
	return sensor
}

func (s *udpSensor) Address() string {
	return s.listener.LocalAddr().String()
}

func (s *udpSensor) Collect(receiver sensor.EventReceiver) error {
	s.logger.Debug("listening")
	for {
		decoder := &log.SyslogDecoder{}
		err := decoder.Read(s.listener)
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			s.logger.Debug("listener closed")
			break
		}
		if err != nil {
			s.logger.Error("read/decode failure", slog.Any("err", err))
			return err
		}
		queueSyslogMessages(s.index, receiver, decoder, s.logger)
	}
	return nil
}

func (s *udpSensor) Shutdown(ctx context.Context) error {
	return nil
}

func (s *udpSensor) Close() error {
	return s.listener.Close()
}

func queueSyslogMessages(index *logmatcher.Index, receiver sensor.EventReceiver, decoder *log.SyslogDecoder, logger *slog.Logger) {
	for _, message := range decoder.Decode() {
		switch message := message.(type) {
		case *log.UndecodedSyslogMessage:
			logger.Warn("undecoded syslog message", slog.String("message", message.String()))
		case *log.RFC3164SyslogMessage:
			queueSyslogMessage(index, receiver, message.Timestamp, message.MessageContent, message.UndecodedSyslogMessage.String())
		case *log.RFC5424SyslogMessage:
			queueSyslogMessage(index, receiver, message.Timestamp, message.Msg, message.UndecodedSyslogMessage.String())
		}
	}
}

func queueSyslogMessage(index *logmatcher.Index, receiver sensor.EventReceiver, timestamp time.Time, message string, source string) {
	resolved := index.ResolveValues(message)
	if resolved != nil {
		event := &sensor.Event{
			Timestamp:       timestamp,
			Type:            resolved.EventType,
			HardwareAddress: resolved.HardwareAddress,
			IPAddress:       resolved.IPAddress,
			User:            resolved.User,
			Service:         resolved.Service,
			Sensor:          Name,
			Source:          source,
		}
		receiver.Queue(event)
	}
}
