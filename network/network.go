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

package network

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"slices"
	"strings"
	"sync"
)

type nameEntry struct {
	name  string
	cidrs []*net.IPNet
}

func (e *nameEntry) Save(w io.Writer) (int, error) {
	total := 0
	for _, cidr := range e.cidrs {
		written, err := w.Write([]byte(fmt.Sprintf("%s:%s\n", e.name, cidr.String())))
		total += written
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (e *nameEntry) Contains(ip net.IP) bool {
	for _, cidr := range e.cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

type Names struct {
	names map[string]*nameEntry
	mutex sync.RWMutex
}

func NewNames() *Names {
	return &Names{
		names: make(map[string]*nameEntry),
	}
}

func (n *Names) Load(r io.Reader) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	lines := bufio.NewReader(r)
	for {
		line, err := lines.ReadString('\n')
		eof := errors.Is(err, io.EOF)
		if err != nil && !eof {
			return err
		}
		line = strings.TrimSpace(line)
		if line != "" {
			name, cidr, err := n.decodeEntryLine(line)
			if err != nil {
				return err
			}
			n.addLocked(name, cidr)
		}
		if eof {
			return nil
		}
	}
}

func (n *Names) decodeEntryLine(line string) (string, *net.IPNet, error) {
	lineSplit := strings.SplitN(line, ":", 2)
	if len(lineSplit) != 2 {
		return "", nil, fmt.Errorf("unrecognized network line: '%s'", line)
	}
	name := strings.TrimSpace(lineSplit[0])
	if name == "" {
		return "", nil, fmt.Errorf("missing network name: '%s'", line)
	}
	_, cidr, err := net.ParseCIDR(strings.TrimSpace(lineSplit[1]))
	if err != nil {
		return "", nil, fmt.Errorf("unrecognized network CIDR: '%s'", line)
	}
	return name, cidr, nil
}

func (n *Names) Save(w io.Writer) (int, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	total := 0
	for _, entry := range n.names {
		written, err := entry.Save(w)
		total += written
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (n *Names) Add(name string, cidrs ...*net.IPNet) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.addLocked(name, cidrs...)
}

func (n *Names) addLocked(name string, cidrs ...*net.IPNet) {
	entry, ok := n.names[name]
	if !ok {
		entry = &nameEntry{
			name:  name,
			cidrs: cidrs,
		}
		n.names[name] = entry
	} else {
		entry.cidrs = append(entry.cidrs, cidrs...)
	}
}

func (n *Names) Names() []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	names := slices.Collect(maps.Keys(n.names))
	slices.Sort(names)
	return names
}

const Unspecified string = "unspecified"
const Loopback string = "loopback"
const LocalMulticast string = "local-multicast"
const Multicast string = "multicast"
const Private string = "private"
const LocalUnicast string = "local-unicast"
const GlobalUnicast string = "global-unicast"

func (n *Names) Match(ip net.IP) string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for name, entry := range n.names {
		if entry.Contains(ip) {
			return name
		}
	}
	if ip.IsUnspecified() {
		return Unspecified
	}
	if ip.IsLoopback() {
		return Loopback
	}
	if ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() {
		return LocalMulticast
	}
	if ip.IsMulticast() {
		return Multicast
	}
	if ip.IsPrivate() {
		return Private
	}
	if ip.IsLinkLocalUnicast() {
		return LocalUnicast
	}
	return GlobalUnicast
}
