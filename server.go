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
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/go-tlsconf/tlsclient"
	"github.com/tdrn-org/netscanner/dns"
	"github.com/tdrn-org/netscanner/internal/datastore"
	"github.com/tdrn-org/netscanner/internal/device"
	"github.com/tdrn-org/netscanner/internal/metrics"
	"github.com/tdrn-org/netscanner/internal/web"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/network"
	"github.com/tdrn-org/netscanner/sensor"
)

type Server struct {
	config          *Config
	httpServer      *httpserver.Instance
	baseURL         *url.URL
	store           *datastore.Store
	metricsRecorder metrics.Recorder
	deviceInfos     *device.InfoCache
	defaultHost     string
	sensors         map[string]*sensor.Sensor
	logMatchers     map[string]*logmatcher.Index
	mutex           sync.RWMutex
	logger          *slog.Logger
}

func StartServer(ctx context.Context, config *Config) (*Server, error) {
	// Setup early logger with configuration address (which may not be the final one).
	// We will reset the logger after listener has been created.
	earlyLogger := slog.With(slog.String("address", config.Server.Address))
	s := &Server{
		config:      config,
		sensors:     map[string]*sensor.Sensor{},
		logMatchers: map[string]*logmatcher.Index{},
		logger:      earlyLogger,
	}
	startFuncs := []func(ctx context.Context, config *Config) error{
		s.startHttpServer,
		s.startDatastore,
		s.startMetrics,
		s.startInfoCache,
		s.startSensors,
	}
	for _, startFunc := range startFuncs {
		err := startFunc(ctx, config)
		if err != nil {
			defer s.Close()
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) startHttpServer(ctx context.Context, config *Config) error {
	s.logger.Info("starting HTTP server...")
	httpServerOptions := config.Server.httpServerOptions()
	httpServer, err := httpserver.Listen(ctx, "tcp", config.Server.Address, httpServerOptions...)
	if err != nil {
		return err
	}
	err = web.MountStatics(httpServer)
	if err != nil {
		return err
	}
	httpServer.HandleFunc("GET /ping", s.handlePingGet)
	s.httpServer = httpServer
	if config.Server.PublicURL.URL != nil {
		s.baseURL = config.Server.PublicURL.URL
	} else {
		s.baseURL = httpServer.BaseURL()
	}
	// Replace early logger by one attributed with actual URL
	s.logger = slog.With(slog.String("baseURL", s.baseURL.String()))
	return nil
}

func (s *Server) handlePingGet(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	err := s.store.Ping(r.Context())
	if err != nil {
		s.logger.Warn("datastore ping failure", slog.Any("err", err))
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
}

func (s *Server) shutdownHttpServer(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) closeHttpServer() error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Close()
}

func (s *Server) startDatastore(ctx context.Context, config *Config) error {
	datastoreConfig, err := config.Datastore.config()
	if err != nil {
		return err
	}
	driver, err := database.Open(datastoreConfig)
	if err != nil {
		return err
	}
	_, _, err = driver.UpdateSchema(ctx)
	if err != nil {
		return errors.Join(err, driver.Close())
	}
	s.store = datastore.New(driver)
	return nil
}

func (s *Server) closeDatastore() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *Server) startMetrics(ctx context.Context, config *Config) error {
	if !config.Metrics.Enabled {
		s.metricsRecorder = metrics.NewRecorder(nil)
		return nil
	}
	s.logger.Info("enabling Metrics endpoint...", slog.String("path", config.Metrics.Path))
	registry := prometheus.NewRegistry()
	if config.Metrics.Process {
		err := registry.Register(collectors.NewGoCollector())
		if err != nil {
			return fmt.Errorf("failed to register process metric collector (cause: %w)", err)
		}
	}
	s.metricsRecorder = metrics.NewRecorder(registry)
	s.httpServer.Handle("GET "+config.Metrics.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return nil
}

func (s *Server) startInfoCache(ctx context.Context, config *Config) error {
	networks := network.NewNames()
	deviceInfos, err := device.NewInfoCache(networks, dns.NewResolverProvider(nil, nil))
	if err != nil {
		return fmt.Errorf("failed to setup device info cache (cause: %w)", err)
	}
	s.deviceInfos = deviceInfos
	return nil
}

func (s *Server) closeInfoCache() error {
	if s.deviceInfos == nil {
		return nil
	}
	return s.deviceInfos.Close()
}

func (s *Server) startSensors(ctx context.Context, config *Config) error {
	s.defaultHost = strings.TrimSpace(config.Sensors.DefaultHost)
	if s.defaultHost == "" {
		host, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to retrieve host name (cause: %w)", err)
		}
		s.defaultHost = host
	}
	for sensorConfig := range config.Sensors.Configs() {
		sensor, err := s.AddSensor(ctx, sensorConfig)
		if err != nil {
			s.logger.Warn("failed to add sensor", slog.Any("sensor", sensorConfig), slog.Any("err", err))
			continue
		}
		go func() {
			err := sensor.Collect(s.eventReceiver())
			if err != nil {
				s.logger.Error("collect failure", slog.Any("sensor", sensor), slog.Any("err", err))
			}
		}()
	}
	return nil
}

func (s *Server) shutdownSensors(ctx context.Context) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	shutdownErrs := make([]error, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		err := sensor.Shutdown(ctx)
		if err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}
	return errors.Join(shutdownErrs...)
}

func (s *Server) closeSensors() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	closeErrs := make([]error, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		err := sensor.Close()
		if err != nil {
			closeErrs = append(closeErrs, err)
		}
	}
	return errors.Join(closeErrs...)
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("serving HTTP requests...")
	err := s.httpServer.Serve()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Ping(ctx context.Context) error {
	if s.httpServer == nil {
		return fmt.Errorf("server not started")
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsclient.GetConfig(),
		},
	}
	pingURL := s.httpServer.BaseURL().JoinPath("/ping").String()
	rsp, err := client.Get(pingURL)
	if err != nil {
		return fmt.Errorf("failed to access URL: '%s' (cause: %w)", pingURL, err)
	}
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to ping URL: '%s' (status: %s)", pingURL, rsp.Status)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return errors.Join(s.Shutdown(ctx), s.Close())
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownFuncs := []func(ctx context.Context) error{
		s.shutdownSensors,
		s.shutdownHttpServer,
	}
	shutdownErrs := make([]error, 0, len(shutdownFuncs))
	for _, shutdownFunc := range shutdownFuncs {
		err := shutdownFunc(ctx)
		if err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}
	return errors.Join(shutdownErrs...)
}

func (s *Server) Close() error {
	closeFuncs := []func() error{
		s.closeSensors,
		s.closeInfoCache,
		s.closeHttpServer,
		s.closeDatastore,
	}
	closeErrs := make([]error, 0, len(closeFuncs))
	for _, closeFunc := range closeFuncs {
		closeErrs = append(closeErrs, closeFunc())
	}
	return errors.Join(closeErrs...)
}
