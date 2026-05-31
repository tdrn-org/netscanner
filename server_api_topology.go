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
	"fmt"
	"sort"
)

// TopologyNode represents a device in the network topology.
type TopologyNode struct {
	ID              string  `json:"id"`
	Label           string  `json:"label"`
	Address         string  `json:"address"`
	HardwareVendor  string  `json:"hardwareVendor,omitempty"`
	HardwareAddress string  `json:"hardwareAddress,omitempty"`
	DNS             string  `json:"dns,omitempty"`
	Network         string  `json:"network"`
	CountryCode     string  `json:"countryCode,omitempty"`
	Lat             float64 `json:"lat"`
	Lng             float64 `json:"lng"`
	NodeType        string  `json:"type"` // "client", "server", "both"
	ConnectionCount int     `json:"connectionCount"`
}

// TopologyEdge represents a connection between two devices.
type TopologyEdge struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Service string `json:"service"`
	Status  string `json:"status"`
	Count   int64  `json:"count"`
}

// Topology represents the complete network topology graph.
type Topology struct {
	Nodes []*TopologyNode `json:"nodes"`
	Edges []*TopologyEdge `json:"edges"`
}

// GetTopology builds a topology graph from recorded connections.
func (s *Server) GetTopology(ctx context.Context) (*Topology, error) {
	page, err := s.ListConnections(ctx, ConnectionQuery{Limit: 0}) // 0 = no limit
	if err != nil {
		return nil, fmt.Errorf("failed to get topology (cause: %w)", err)
	}
	connections := page.Items

	// Build node map (deduplicate by address)
	nodeMap := make(map[string]*TopologyNode)
	edgeMap := make(map[string]*TopologyEdge) // key: "source:target:service"

	for _, conn := range connections {
		// Client node
		clientKey := conn.Client.Address
		if _, ok := nodeMap[clientKey]; !ok {
			label := conn.Client.DNS
			if label == "" {
				label = conn.Client.Address
			}
			nodeMap[clientKey] = &TopologyNode{
				ID:              clientKey,
				Label:           label,
				Address:         conn.Client.Address,
				HardwareVendor:  conn.Client.HardwareVendor,
				HardwareAddress: conn.Client.HardwareAddress,
				DNS:             conn.Client.DNS,
				Network:         conn.Client.Network,
				CountryCode:     conn.Client.CountryCode,
				Lat:             conn.Client.Lat,
				Lng:             conn.Client.Lng,
				NodeType:        "client",
			}
		}

		// Server node
		serverKey := conn.Server.Address
		if _, ok := nodeMap[serverKey]; !ok {
			label := conn.Server.DNS
			if label == "" {
				label = conn.Server.Address
			}
			nodeMap[serverKey] = &TopologyNode{
				ID:              serverKey,
				Label:           label,
				Address:         conn.Server.Address,
				HardwareVendor:  conn.Server.HardwareVendor,
				HardwareAddress: conn.Server.HardwareAddress,
				DNS:             conn.Server.DNS,
				Network:         conn.Server.Network,
				CountryCode:     conn.Server.CountryCode,
				Lat:             conn.Server.Lat,
				Lng:             conn.Server.Lng,
				NodeType:        "server",
			}
		}

		// If a node appears as both client and server, mark it
		if nodeMap[clientKey].NodeType == "server" || nodeMap[serverKey].NodeType == "client" {
			nodeMap[clientKey].NodeType = "both"
			nodeMap[serverKey].NodeType = "both"
		}

		// Increment connection counts
		nodeMap[clientKey].ConnectionCount++
		nodeMap[serverKey].ConnectionCount++

		// Edge
		edgeKey := fmt.Sprintf("%s:%s:%s", clientKey, serverKey, conn.Service)
		if existing, ok := edgeMap[edgeKey]; ok {
			existing.Count += conn.Count
		} else {
			edgeMap[edgeKey] = &TopologyEdge{
				Source:  clientKey,
				Target:  serverKey,
				Service: conn.Service,
				Status:  conn.Status,
				Count:   conn.Count,
			}
		}
	}

	// Convert maps to sorted slices
	nodes := make([]*TopologyNode, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ConnectionCount > nodes[j].ConnectionCount
	})

	edges := make([]*TopologyEdge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, edge)
	}

	return &Topology{Nodes: nodes, Edges: edges}, nil
}
