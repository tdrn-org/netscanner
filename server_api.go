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
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	_ "github.com/tdrn-org/netscanner/api"
)

//	@title			NetScanner REST API
//	@version		1.0
//	@description	Network activity monitoring server.

//	@contact.url	https://github.com/tdrn-org/netscanner

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:9123
//	@BasePath	/api/v1

// GET @BasePath/ping
//
//	@Summary		Ping the server
//	@Description	Ping the server to check general health
//	@Produce		text/plain
//	@Success		200	{string}	string	"ok"
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/ping [get]
func (s *Server) handlePingGet(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	err := s.store.Ping(r.Context())
	if err != nil {
		s.logger.Warn("datastore ping failure", slog.Any("err", err))
		status = http.StatusInternalServerError
	}
	if status != http.StatusOK {
		s.sendAPIError(w, r, http.StatusInternalServerError, nil)
		return
	}
	s.sendAPIPlainTextResponse(w, r, http.StatusOK, "ok")
}

// SensorInfo represents a running sensor including its stats.
type SensorInfo struct {
	// The unique name of the sensor
	Name string `json:"name"`
	// The type of the sensor (syslog, accesslog, ...)
	Type string `json:"type"`
	// The number of events the sensor has collected since it has been started
	EventCounter uint64 `json:"event_counter"`
}

// GET @BasePath/sensor
//
//	@Summary		List sensors
//	@Description	List all running sensors as well as their stats
//	@Produce		json
//	@Success		200	{object}	[]SensorInfo
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/sensor [get]
func (s *Server) handleSensorsGet(w http.ResponseWriter, r *http.Request) {
	sensorInfos := s.ListSensors()
	s.sendAPIApplicationJSONResponse(w, r, http.StatusOK, sensorInfos)
}

// GET @BasePath/rules/lmi
//
//	@Summary		List log matcher index names
//	@Description	List the names available log matcher indexes
//	@Produce		json
//	@Success		200	{object}	[]string
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/rules/lmi [get]
func (s *Server) handleLMIsGet(w http.ResponseWriter, r *http.Request) {
	names, err := s.ListLogMatcherIndexNames(r.Context())
	if err != nil {
		s.sendAPIError(w, r, http.StatusInternalServerError, err)
		return
	}
	s.sendAPIApplicationJSONResponse(w, r, http.StatusOK, names)
}

// DeviceInfo represents a network device (server or client)
type DeviceInfo struct {
	// The ID of the device
	ID string `json:"id"`
	// The network address of the device
	Address string `json:"address"`
	// The network the address belongs to
	Network string `json:"network"`
	// The hardware address of the device (if available)
	HardwareAddress string `json:"hardware_address"`
	// The hardware vendor of the device (derived from hardware address)
	HardwareVendor string `json:"hardware_vendor"`
	// The DNS name of the device (if available)
	DNS string `json:"dns"`
	// The latitude of the device's location (if available)
	Lat float64 `json:"lat"`
	// The longitude of the device's location (if available)
	Lng float64 `json:"lng"`
	// The city of the device's location (if available)
	City string `json:"city"`
	// The country of the device's location (if available)
	Country string `json:"country"`
	// The country code of the device's location (if available)
	CountryCode string `json:"country_code"`
}

// GET @BasePath/device/{id}
//
//	@Summary		Get device info
//	@Description	Get device info for the given ID
//	@Produce		json
//	@Produce		text/plain
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{object}	DeviceInfo
//	@Failure		404	{string}	string	"not found"
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/device/{id} [get]
func (s *Server) handleDeviceGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	deviceInfo, err := s.GetDevice(r.Context(), id)
	if errors.Is(err, ErrDeviceNotFound) {
		s.sendAPIPlainTextResponse(w, r, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		s.sendAPIError(w, r, http.StatusInternalServerError, err)
	}
	err = json.NewEncoder(w).Encode(deviceInfo)
	if err != nil {
		s.sendAPIError(w, r, http.StatusInternalServerError, err)
		return
	}
}

// ConnectionInfo represents a logged network connection.
type ConnectionInfo struct {
	// The ID of the connection
	ID string `json:"id"`
	// The device info of the connected server
	Server DeviceInfo `json:"server"`
	// The accessed service
	Service string `json:"service"`
	// The device info of the connecting client
	Client DeviceInfo `json:"client"`
	// The status of connection (granted, denied, error, informational)
	Status string `json:"status"`
	// The number how often this connection has been logged
	Count int `json:"count"`
	// The first point in time this connection has been logged
	First int `json:"first"`
	// The last point in time this connection has been logged
	Last int `json:"last"`
}

// GET @BasePath/connection
//
//	@Summary		List logged connections
//	@Description	List logged connections
//	@Produce		json
//	@Produce		text/plain
//	@Success		200	{object}	[]ConnectionInfo
//	@Failure		500	{string}	string	"nok"
//	@Router			/api/v1/connection [get]
func (s *Server) handleConnectionsGet(w http.ResponseWriter, r *http.Request) {
	conncetionInfos, err := s.ListConnections(r.Context())
	if err != nil {
		s.sendAPIError(w, r, http.StatusInternalServerError, err)
		return
	}
	err = json.NewEncoder(w).Encode(conncetionInfos)
	if err != nil {
		s.sendAPIError(w, r, http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) sendAPIApplicationJSONResponse(w http.ResponseWriter, r *http.Request, status int, content any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(content)
	if err != nil {
		s.logger.Error("failed to send 'application/json' response", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
	}
}

func (s *Server) sendAPIPlainTextResponse(w http.ResponseWriter, r *http.Request, status int, content string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, err := w.Write([]byte(content))
	if err != nil {
		s.logger.Error("failed to send 'text/plain' response", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
	}
}

func (s *Server) sendAPIError(w http.ResponseWriter, r *http.Request, status int, cause error) {
	if cause != nil {
		s.logger.Error("http handler failure", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", cause))
	}
	http.Error(w, "server error", status)
}
