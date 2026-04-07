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
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/netip"
	"os/exec"
	"reflect"
	"strings"
)

type NMapRun struct {
	XMLName          xml.Name     `xml:"nmaprun"`
	Scanner          string       `xml:"scanner,attr"`
	Args             string       `xml:"args,attr"`
	Start            int64        `xml:"start,attr"`
	Version          string       `xml:"version,attr"`
	XMLOutputVersion string       `xml:"xmloutputversion,attr"`
	ScanInfo         NMapScanInfo `xml:"scaninfo"`
	Hosts            []NMapHost   `xml:"host"`
}

type NMapScanInfo struct {
	XMLName  xml.Name `xml:"scaninfo"`
	Type     string   `xml:"type,attr"`
	Protocol string   `xml:"protocol,attr"`
}

type NMapHost struct {
	XMLName   xml.Name        `xml:"host"`
	StartTime int64           `xml:"starttime,attr"`
	EndTime   int64           `xml:"endtime,attr"`
	Status    NMapHostStatus  `xml:"status"`
	Address   NMapHostAddress `xml:"address"`
	Hostnames NMapHostnames   `xml:"hostnames"`
	Ports     NMapPorts       `xml:"ports"`
	OS        NMapOS          `xml:"os"`
}

type NMapHostStatus struct {
	XMLName xml.Name `xml:"status"`
	State   string   `xml:"state,attr"`
	Reason  string   `xml:"reason,attr"`
}

type NMapHostAddress struct {
	XMLName  xml.Name `xml:"address"`
	Addr     string   `xml:"addr,attr"`
	AddrType string   `xml:"addrtype,attr"`
}

type NMapHostnames struct {
	XMLName  xml.Name       `xml:"hostnames"`
	Elements []NMapHostname `xml:"hostname"`
}

type NMapHostname struct {
	XMLName xml.Name `xml:"hostname"`
	Name    string   `xml:"name,attr"`
	Type    string   `xml:"type,attr"`
}

type NMapPorts struct {
	XMLName  xml.Name   `xml:"ports"`
	Elements []NMapPort `xml:"port"`
}

type NMapPort struct {
	XMLName  xml.Name      `xml:"port"`
	Protocol string        `xml:"protocol,attr"`
	PortID   int           `xml:"portid,attr"`
	State    NMapPortState `xml:"state"`
	Service  *NMapService  `xml:"service"`
}

type NMapPortState struct {
	XMLName xml.Name `xml:"state"`
	State   string   `xml:"state,attr"`
	Reason  string   `xml:"reason,attr"`
}

type NMapService struct {
	XMLName xml.Name `xml:"service"`
	Name    string   `xml:"name,attr"`
}

type NMapOS struct {
	XMLName xml.Name      `xml:"os"`
	OSClass []NMapOSClass `xml:"osclass"`
	OSMatch []NMapOSMatch `xml:"osmatch"`
}

type NMapOSClass struct {
	XMLName  xml.Name `xml:"osclass"`
	Type     string   `xml:"type,attr"`
	Vendor   string   `xml:"vendor,attr"`
	OSGen    string   `xml:"osgen,attr"`
	Accuracy int      `xml:"accuracy,attr"`
}

type NMapOSMatch struct {
	XMLName  xml.Name `xml:"osmatch"`
	Name     string   `xml:"name,attr"`
	Accuracy int      `xml:"accuracy,attr"`
}

type NMapResult struct {
	address netip.Addr
	err     error
	Run     *NMapRun
}

func (r *NMapResult) Address() netip.Addr {
	return r.address
}

func (r *NMapResult) Error() error {
	return r.err
}

func (r *NMapResult) String() string {
	buffer := &strings.Builder{}
	formatResult(buffer, r)
	ports, services := 0, 0
	if r.Run != nil {
		host := r.Run.Hosts[0]
		for _, port := range host.Ports.Elements {
			ports++
			if port.Service != nil {
				services++
			}
		}
	}
	fmt.Fprintf(buffer, " (ports: %d services: %d)", ports, services)
	return buffer.String()
}

func (r *NMapResult) Up() bool {
	return r.Run != nil && r.Run.Hosts[0].Status.State == "up"
}

const DefaultNMapCommand string = "nmap"
const DefaultNMapArg string = "-sT"

type NMap struct {
	Command string
	Args    []string
	logger  *slog.Logger
}

func NewNMap() *NMap {
	return &NMap{
		Command: DefaultNMapCommand,
		Args:    []string{DefaultNMapArg},
		logger:  slog.With(slog.String("probe", reflect.TypeFor[NMap]().String())),
	}
}

func (p *NMap) Run(ctx context.Context, address netip.Addr) *NMapResult {
	runLogger := p.logger.With(slog.Any("address", address))
	runLogger.Info("running...")
	commandArgs := p.buildArgs(address)
	cmd := exec.CommandContext(ctx, p.Command, commandArgs...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return &NMapResult{
			address: address,
			err:     err,
		}
	}
	defer stdoutPipe.Close()
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return &NMapResult{
			address: address,
			err:     err,
		}
	}
	defer stderrPipe.Close()
	runLogger.Debug("running command", slog.Any("cmd", cmd))
	err = cmd.Start()
	if err != nil {
		return &NMapResult{
			address: address,
			err:     err,
		}
	}
	run := &NMapRun{}
	go func() {
		decoder := xml.NewDecoder(stdoutPipe)
		err := decoder.Decode(run)
		if err != nil {
			runLogger.Warn("failed to parse nmap output", slog.Any("err", err))
		}
	}()
	go func() {
		buffer := bufio.NewReader(stderrPipe)
		for {
			line, _, err := buffer.ReadLine()
			if err != nil {
				return
			}
			runLogger.Error(string(line))
		}
	}()
	err = cmd.Wait()
	result := &NMapResult{
		address: address,
		err:     err,
		Run:     run,
	}
	runLogger.Info("completed", slog.Any("err", result.err))
	return result
}

func (p *NMap) buildArgs(address netip.Addr) []string {
	commandArgs := make([]string, 0, len(p.Args)+4)
	commandArgs = append(commandArgs, p.Args...)
	commandArgs = append(commandArgs, "-oX", "-")
	if address.Is6() {
		commandArgs = append(commandArgs, "-6")
	}
	commandArgs = append(commandArgs, address.String())
	return commandArgs
}
