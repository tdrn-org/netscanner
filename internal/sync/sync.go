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

package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"

	"github.com/tdrn-org/netscanner/internal/mtls"
	"github.com/tdrn-org/netscanner/internal/sync/proto"
	"github.com/tdrn-org/netscanner/sensor"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler interface {
	sensor.EventReceiver
	Shutdown(ctx context.Context) error
	Close() error
}

func StartReceive(address string, credentials *mtls.Credentials, receiver sensor.EventReceiver) (Handler, error) {
	tlsConfig, err := credentials.TLSConfig()
	if err != nil {
		return nil, err
	}
	grpcCredentials := grpccredentials.NewTLS(tlsConfig)
	server := grpc.NewServer(grpc.Creds(grpcCredentials))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on address '%s' (cause: %w)", address, err)
	}
	logger := slog.With(slog.String("sync", "receive"), slog.String("address", listener.Addr().String()))
	handler := &receiveHandler{
		server:   server,
		listener: listener,
		receiver: receiver,
		logger:   logger,
	}
	proto.RegisterEventStreamerServer(server, handler)
	go handler.Serve()
	return handler, nil
}

type receiveHandler struct {
	proto.UnimplementedEventStreamerServer
	server   *grpc.Server
	listener net.Listener
	receiver sensor.EventReceiver
	logger   *slog.Logger
}

func (h *receiveHandler) Serve() {
	h.logger.Info("serving...")
	err := h.server.Serve(h.listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		h.logger.Info("stopped")
	} else if err != nil {
		h.logger.Error("serve failure", slog.Any("err", err))
	}
}

var sensorEventTypeMap map[proto.EventType]sensor.EventType = map[proto.EventType]sensor.EventType{
	proto.EventType_EVENT_TYPE_INFORMATIONAL: sensor.EventTypeInformational,
	proto.EventType_EVENT_TYPE_GRANTED:       sensor.EventTypeGranted,
	proto.EventType_EVENT_TYPE_DENIED:        sensor.EventTypeDenied,
	proto.EventType_EVENT_TYPE_ERROR:         sensor.EventTypeError,
}

func (h *receiveHandler) SendEvent(ctx context.Context, event *proto.Event) (*proto.EmptyResponse, error) {
	sensorEventType, ok := sensorEventTypeMap[event.Type]
	if !ok {
		return &proto.EmptyResponse{}, fmt.Errorf("unrecognized event type: %s", event.Type)
	}
	var address netip.Addr
	switch len(event.Address) {
	case 4:
		address = netip.AddrFrom4([4]byte(event.Address))
	case 16:
		address = netip.AddrFrom16([16]byte(event.Address))
	default:
		return &proto.EmptyResponse{}, fmt.Errorf("unexpected address lenght: %d", len(event.Address))
	}
	sensorEvent := &sensor.Event{
		Host:            event.Host,
		Timestamp:       event.Timestamp.AsTime(),
		Type:            sensorEventType,
		Address:         address,
		HardwareAddress: event.HardwareAddress,
		User:            event.User,
		Service:         event.Service,
		Sensor:          event.Sensor,
	}
	h.receiver.Queue(ctx, sensorEvent)
	return &proto.EmptyResponse{}, nil
}

func (h *receiveHandler) Queue(ctx context.Context, event *sensor.Event) {
	// Nothing to do here
}

func (h *receiveHandler) Shutdown(ctx context.Context) error {
	ch := make(chan struct{})
	go func() {
		h.server.GracefulStop()
		close(ch)
	}()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		h.Close()
		return ctx.Err()
	}
}

func (h *receiveHandler) Close() error {
	h.server.Stop()
	return nil
}

func StartForward(address string, credentials *mtls.Credentials) (Handler, error) {
	tlsConfig, err := credentials.TLSConfig()
	if err != nil {
		return nil, err
	}
	grpcCredentials := grpccredentials.NewTLS(tlsConfig)
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(grpcCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to start sync client (cause: %w)", err)
	}
	logger := slog.With(slog.String("sync", "forward"), slog.String("address", address))
	handler := &forwardHandler{
		conn:   conn,
		client: proto.NewEventStreamerClient(conn),
		logger: logger,
	}
	return handler, nil
}

type forwardHandler struct {
	conn   *grpc.ClientConn
	client proto.EventStreamerClient
	logger *slog.Logger
}

var protoEventTypeMap map[sensor.EventType]proto.EventType = map[sensor.EventType]proto.EventType{
	sensor.EventTypeInformational: proto.EventType_EVENT_TYPE_INFORMATIONAL,
	sensor.EventTypeGranted:       proto.EventType_EVENT_TYPE_GRANTED,
	sensor.EventTypeDenied:        proto.EventType_EVENT_TYPE_DENIED,
	sensor.EventTypeError:         proto.EventType_EVENT_TYPE_ERROR,
}

func (h *forwardHandler) Queue(ctx context.Context, event *sensor.Event) {
	protoEventType, ok := protoEventTypeMap[event.Type]
	if !ok {
		h.logger.Warn("unrecognized event type", slog.String("type", string(event.Type)))
		return
	}
	req := &proto.Event{
		Host:            event.Host,
		Timestamp:       timestamppb.New(event.Timestamp),
		Type:            protoEventType,
		Address:         event.Address.AsSlice(),
		HardwareAddress: event.HardwareAddress,
		User:            event.User,
		Service:         event.Service,
		Sensor:          event.Sensor,
	}
	_, err := h.client.SendEvent(ctx, req)
	if err != nil {
		h.logger.Warn("failed to send event", slog.Any("err", err))
		return
	}
}

func (h *forwardHandler) Shutdown(_ context.Context) error {
	return nil
}

func (h *forwardHandler) Close() error {
	return h.conn.Close()
}
