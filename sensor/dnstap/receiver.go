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
	"net/netip"
	"slices"
	"time"

	"codeberg.org/miekg/dns"
)

const DefaultMaxFrameSize int = 96 * 1024

type Receiver interface {
	Consume(consum EntryConsumer)
	Shutdown(ctx context.Context) error
	Close() error
}

type EntryConsumer func(entry *Entry)

type Entry struct {
	Content    *Dnstap
	skipBefore time.Time
}

func (e *Entry) Decode() (netip.Addr, netip.Addr, bool) {
	message, ok := e.decodeMessage()
	if !ok {
		return netip.Addr{}, netip.Addr{}, false
	}
	clientAddress, ok := e.decodeClientAddress(message)
	if !ok {
		return netip.Addr{}, netip.Addr{}, false
	}
	serverAddress, ok := e.decodeServerAddress(message)
	if !ok {
		return netip.Addr{}, netip.Addr{}, false
	}
	return clientAddress, serverAddress, true
}

func (e *Entry) decodeMessage() (*Message, bool) {
	message := e.Content.GetMessage()
	if message == nil {
		return nil, false
	}
	messageType := message.GetType()
	if messageType != Message_CLIENT_RESPONSE && messageType != Message_AUTH_RESPONSE {
		return nil, false
	}
	if message.GetQueryTimeSec() < uint64(e.skipBefore.Unix()) {
		return nil, false
	}
	return message, true
}

func (e *Entry) decodeClientAddress(message *Message) (netip.Addr, bool) {
	addressBytes := message.GetQueryAddress()
	if len(addressBytes) == 0 {
		return netip.Addr{}, false
	}
	return netip.AddrFromSlice(addressBytes)
}

func (e *Entry) decodeServerAddress(message *Message) (netip.Addr, bool) {
	responseBytes := message.GetResponseMessage()
	if len(responseBytes) == 0 {
		return netip.Addr{}, false
	}
	response := &dns.Msg{Data: responseBytes}
	err := response.Unpack()
	if err != nil {
		return netip.Addr{}, false
	}
	if response.Rcode != dns.RcodeSuccess || len(response.Question) == 0 {
		return netip.Addr{}, false
	}
	qType := dns.RRToType(response.Question[0])
	if qType != dns.TypeA && qType != dns.TypeAAAA {
		return netip.Addr{}, false
	}
	return e.decodeResponseAnswer(response.Answer, qType)
}

func (e *Entry) decodeResponseAnswer(answer []dns.RR, qType uint16) (netip.Addr, bool) {
	resolvedIPs := make([]string, 0, len(answer))
	for _, answer := range answer {
		switch rr := answer.(type) {
		case *dns.A:
			if qType == dns.TypeA {
				resolvedIPs = append(resolvedIPs, rr.A.String())
			}
		case *dns.AAAA:
			if qType == dns.TypeAAAA {
				resolvedIPs = append(resolvedIPs, rr.AAAA.String())
			}
		}
	}
	slices.Sort(resolvedIPs)
	for _, resolvedIP := range resolvedIPs {
		address, err := netip.ParseAddr(resolvedIP)
		if err != nil {
			continue
		}
		return address, true
	}
	return netip.Addr{}, false
}
