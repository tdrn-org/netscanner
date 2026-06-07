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

package dnstap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	framestream "github.com/farsightsec/golang-framestream"
	"google.golang.org/protobuf/proto"
)

type socketReceiver struct {
	path         string
	listener     *net.UnixListener
	maxFrameSize int
	skipBefore   time.Time
	logger       *slog.Logger
	stopping     atomic.Bool
	waitStopped  sync.WaitGroup
	mutex        sync.Mutex
	activeConns  map[net.Conn]net.Conn
}

func NewSocketReceiver(path string, mode os.FileMode, maxFrameSize int, tail bool) (Receiver, error) {
	logger := slog.With(slog.String("sensor", Name), slog.String("socket", path))
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to remove stall socket '%s' (cause: %w)", path, err)
	}
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve unix address '%s' (cause: %w)", path, err)
	}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on unix address '%s' (cause: %w)", addr.Name, err)
	}
	err = os.Chmod(path, mode)
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to chmod unix socket '%s' (cause: %w)", path, err)
	}
	skipBefore := time.Unix(0, 0)
	if tail {
		skipBefore = time.Now()
	}
	receiver := &socketReceiver{
		path:         path,
		listener:     listener,
		maxFrameSize: maxFrameSize,
		skipBefore:   skipBefore,
		logger:       logger,
		activeConns:  make(map[net.Conn]net.Conn),
	}
	return receiver, nil
}

func (r *socketReceiver) Consume(consumer EntryConsumer) {
	for {
		conn, err := r.listener.AcceptUnix()
		if err != nil {
			break
		}
		r.waitStopped.Go(func() {
			r.handleConn(conn, consumer)
		})
	}
}

const socketReadDeadline time.Duration = 1 * time.Second

func (r *socketReceiver) handleConn(conn *net.UnixConn, consumer EntryConsumer) {
	defer conn.Close()

	if r.stopping.Load() {
		return
	}

	r.logger.Info("new dnstap connection")
	r.registerConn(conn)
	defer r.deregisterConn(conn)
	frameReader, err := framestream.NewReader(conn, &framestream.ReaderOptions{
		ContentTypes:  [][]byte{[]byte("protobuf:dnstap.Dnstap")},
		Bidirectional: false,
	})
	if err != nil {
		r.logger.Error("failed to create frame reader", slog.Any("err", err))
		return
	}
	buffer := make([]byte, r.maxFrameSize)
	for {
		if r.stopping.Load() {
			r.logger.Info("closing connection")
			return
		}
		conn.SetReadDeadline(time.Now().Add(socketReadDeadline))
		frameLen, err := frameReader.ReadFrame(buffer)
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			continue
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			break
		} else if err != nil {
			r.logger.Error("connection read failure", slog.Any("err", err))
			break
		}
		content := &Dnstap{}
		err = proto.Unmarshal(buffer[:frameLen], content)
		if err != nil {
			continue
		}
		entry := &Entry{
			Content:    content,
			skipBefore: r.skipBefore,
		}
		r.logger.Debug("consuming dnstap event")
		consumer(entry)
	}
}

func (r *socketReceiver) registerConn(conn *net.UnixConn) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.activeConns[conn] = conn
}

func (r *socketReceiver) deregisterConn(conn *net.UnixConn) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.activeConns, conn)
}

func (r *socketReceiver) closeConns() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, activeConn := range r.activeConns {
		activeConn.Close()
	}
}

func (r *socketReceiver) Shutdown(ctx context.Context) error {
	r.stopping.Store(true)

	err := r.listener.Close()
	if err != nil {
		r.logger.Warn("failed to close listener", slog.Any("err", err))
	}
	stopped := make(chan bool)
	go func() {
		r.waitStopped.Wait()
		close(stopped)
	}()
	select {
	case <-ctx.Done():
		r.closeConns()
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

func (r *socketReceiver) Close() error {
	r.stopping.Store(true)
	r.listener.Close()
	r.closeConns()
	err := os.Remove(r.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
