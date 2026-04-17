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
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-httpserver"
	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/internal/metrics"
	"github.com/tdrn-org/netscanner/sensor"
)

var ErrUnknownProfile error = errors.New("unknown profile")

type Server struct {
	config          *Config
	httpServer      *httpserver.Instance
	baseURL         *url.URL
	datastore       *database.Driver
	metricsRecorder metrics.Recorder
	activeProfile   ServerProfile
	mutex           sync.RWMutex
	logger          *slog.Logger
}

func StartServer(ctx context.Context, config *Config) (*Server, error) {
	// Setup early logger with configuration address (which may not be the final one).
	// We will reset the logger after listener has been created.
	earlyLogger := slog.With(slog.String("address", config.Server.Address))
	s := &Server{
		config: config,
		logger: earlyLogger,
	}
	startFuncs := []func(ctx context.Context, config *Config) error{
		s.startHttpServer,
		s.startDatastore,
		s.startMetrics,
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
	httpServer.HandleFunc("GET /ping", s.handlePingGet)
	s.httpServer = httpServer
	if config.Server.PublicURL.URL != nil {
		s.baseURL = config.Server.PublicURL.URL
	} else {
		s.baseURL = httpServer.BaseURL()
	}
	// Replace early logger by one attributed by actual URL
	s.logger = slog.With(slog.String("baseURL", s.baseURL.String()))
	return nil
}

func (s *Server) handlePingGet(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	err := s.datastore.Ping(r.Context())
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
	datastore, err := database.Open(datastoreConfig)
	if err != nil {
		return err
	}
	_, _, err = datastore.UpdateSchema(ctx)
	if err != nil {
		return errors.Join(err, datastore.Close())
	}
	s.datastore = datastore
	return nil
}

func (s *Server) closeDatastore() error {
	if s.datastore == nil {
		return nil
	}
	return s.datastore.Close()
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

func (s *Server) Run(ctx context.Context, profileName string) error {
	if profileName != "" {
		profile, err := s.GetProfile(ctx, profileName)
		if err == nil {
			s.logger.Info("activating profile", slog.String("profile", profileName))
			s.activeProfile = profile
			s.activeProfile.Start(ctx)
		} else {
			s.logger.Warn("failed to activate profile", slog.String("profile", profileName), slog.Any("err", err))
		}
	}
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
	insecureSkipVerify := true
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
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
		s.shutdownActiveProfile,
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

func (s *Server) shutdownActiveProfile(ctx context.Context) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.shutdownActiveProfileLocked(ctx)
}

func (s *Server) shutdownActiveProfileLocked(ctx context.Context) error {
	if s.activeProfile == nil {
		return nil
	}
	return s.activeProfile.Shutdown(ctx)
}

func (s *Server) Close() error {
	closeFuncs := []func() error{
		s.closeActiveProfile,
		s.closeHttpServer,
		s.closeDatastore,
	}
	closeErrs := make([]error, 0, len(closeFuncs))
	for _, closeFunc := range closeFuncs {
		closeErrs = append(closeErrs, closeFunc())
	}
	return errors.Join(closeErrs...)
}

func (s *Server) closeActiveProfile() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.closeProfilesLocked()
}

func (s *Server) closeProfilesLocked() error {
	if s.activeProfile == nil {
		return nil
	}
	err := s.activeProfile.Close()
	s.activeProfile = nil
	return err
}

func (s *Server) AddProfile(ctx context.Context, profile *Profile) error {
	txCtx, tx, err := s.datastore.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.EndTx(txCtx)

	err = model.DeleteProfileByName(ctx, s.datastore, profile.Name)
	if err != nil {
		return err
	}
	err = profile.toModel(s.datastore).Insert(txCtx)
	if err != nil {
		return err
	}

	err = tx.CommitTx(txCtx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) GetProfileNames(ctx context.Context) ([]string, error) {
	return model.SelectProfileNames(ctx, s.datastore)
}

func (s *Server) GetProfile(ctx context.Context, name string) (ServerProfile, error) {
	model, err := model.SelectProfileByName(ctx, s.datastore, name)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, ErrUnknownProfile
	}
	profile := &serverProfile{
		receiver: sensor.EventReceiverFunc(s.queueEvent),
		model:    model,
		logger:   s.logger.With(slog.String("profile", model.Name)),
	}
	return profile, nil
}

func (s *Server) queueEvent(event *sensor.Event) {
	s.logger.Info(event.String())
}
