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

package probe

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"reflect"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

type PingResult struct {
	address               netip.Addr
	err                   error
	PacketsRecv           int
	PacketsSent           int
	PacketsRecvDuplicates int
	PacketLoss            float64
	Rtts                  []time.Duration
	TTLs                  []uint8
	MinRtt                time.Duration
	MaxRtt                time.Duration
	AvgRtt                time.Duration
	StdDevRtt             time.Duration
}

func (r *PingResult) Address() netip.Addr {
	return r.address
}

func (r *PingResult) Error() error {
	return r.err
}

func (r *PingResult) Up() bool {
	return r.PacketsRecv > 0
}

func (r *PingResult) String() string {
	buffer := &strings.Builder{}
	formatResult(buffer, r)
	fmt.Fprintf(buffer, " (sent: %d recv: %d lost: %v%%)", r.PacketsSent, r.PacketsRecv, r.PacketLoss)
	return buffer.String()
}

const DefaultPingInterval time.Duration = 1 * time.Second
const DefaultPingCount int = 3

type Ping struct {
	Interface string
	Interval  time.Duration
	Size      int
	TTL       int
	Count     int
	logger    *slog.Logger
}

func NewPing() *Ping {
	return &Ping{
		Interval: DefaultPingInterval,
		Count:    DefaultPingCount,
		logger:   slog.With(slog.String("probe", reflect.TypeFor[Ping]().String())),
	}
}

func (p *Ping) Run(ctx context.Context, address netip.Addr) *PingResult {
	runLogger := p.logger.With(slog.Any("address", address))
	runLogger.Info("running...")
	pinger, err := probing.NewPinger(address.String())
	if err != nil {
		return &PingResult{
			address: address,
			err:     err,
		}
	}
	if p.Interface != "" {
		pinger.InterfaceName = p.Interface
	}
	pinger.Interval = DefaultPingInterval
	if p.Interval > 0 {
		pinger.Interval = p.Interval
	}
	if p.Size > 0 {
		pinger.Size = p.Size
	}
	if p.TTL > 0 {
		pinger.TTL = p.TTL
	}
	pinger.Count = DefaultPingCount
	if p.Count > 0 {
		pinger.Count = p.Count
	}
	pinger.SetLogger(wrappedLogger{Logger: runLogger})
	err = pinger.RunWithContext(ctx)
	statistics := pinger.Statistics()
	result := &PingResult{
		address:               address,
		err:                   err,
		PacketsRecv:           statistics.PacketsRecv,
		PacketsSent:           statistics.PacketsSent,
		PacketsRecvDuplicates: statistics.PacketsRecvDuplicates,
		PacketLoss:            statistics.PacketLoss,
		Rtts:                  statistics.Rtts,
		TTLs:                  statistics.TTLs,
		MinRtt:                statistics.MaxRtt,
		MaxRtt:                statistics.MaxRtt,
		AvgRtt:                statistics.AvgRtt,
		StdDevRtt:             statistics.StdDevRtt,
	}
	runLogger.Info("completed", slog.Any("err", result.err))
	return result
}

type wrappedLogger struct {
	Logger *slog.Logger
}

func (l wrappedLogger) Fatalf(format string, v ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, v...))
}

func (l wrappedLogger) Errorf(format string, v ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, v...))
}

func (l wrappedLogger) Warnf(format string, v ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, v...))
}

func (l wrappedLogger) Infof(format string, v ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, v...))
}

func (l wrappedLogger) Debugf(format string, v ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, v...))
}
