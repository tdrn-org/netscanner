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
	"net"

	"github.com/tdrn-org/netscanner/mtls"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
)

type Handler interface {
	Shutdown(ctx context.Context) error
	Close() error
}

func StartReceive(address string, credentials *mtls.Credentials) (Handler, error) {
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
	handler := &receiveHandler{
		server:   server,
		listener: listener,
	}
	go handler.Serve()
	return handler, nil
}

type receiveHandler struct {
	server   *grpc.Server
	listener net.Listener
}

func (h *receiveHandler) Serve() {
	err := h.server.Serve(h.listener)
	if errors.Is(err, grpc.ErrServerStopped) {

	} else if err != nil {

	}
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
	handler := &forwardHandler{
		conn: conn,
	}
	return handler, nil
}

type forwardHandler struct {
	conn *grpc.ClientConn
}

func (h *forwardHandler) Shutdown(_ context.Context) error {
	return nil
}

func (h *forwardHandler) Close() error {
	return h.conn.Close()
}
