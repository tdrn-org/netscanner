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

import "context"

// TODO: Support filter, sort and pagination
func (s *Server) ListConnections(ctx context.Context) ([]*ConnectionInfo, error) {
	connections, err := s.store.SelectConnectionsByCursor(ctx)
	if err != nil {
		return nil, err
	}
	connectionInfos := make([]*ConnectionInfo, 0, len(connections))
	for _, connection := range connections {
		connectionInfo := &ConnectionInfo{
			ID:      connection.ID,
			Server:  *s.deviceToDeviceInfo(ctx, connection.Server),
			Client:  *s.deviceToDeviceInfo(ctx, connection.Client),
			Service: connection.Service,
			Status:  string(connection.Status),
			Count:   connection.Count,
			First:   connection.First,
			Last:    connection.Last,
		}
		connectionInfos = append(connectionInfos, connectionInfo)
	}
	return connectionInfos, nil
}
