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

package logmatcher

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/tdrn-org/netscanner/sensor"
)

type Index struct {
	name       string
	tokenizer  Tokenizer
	rootNode   indexNode
	matchCount int
	mutex      sync.RWMutex
	logger     *slog.Logger
}

func NewIndex(name string, tokenizer Tokenizer) *Index {
	return &Index{
		name:      name,
		tokenizer: tokenizer,
		logger:    slog.With(slog.String(reflect.TypeFor[Index]().String(), name)),
	}
}

func (i *Index) Load(r io.Reader) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	lines := bufio.NewReader(r)
	for {
		line, err := lines.ReadString('\n')
		eof := errors.Is(err, io.EOF)
		if err != nil && !eof {
			return err
		}
		line = strings.TrimSpace(line)
		if line != "" {
			lineSplit := strings.SplitN(line, ":", 3)
			if len(lineSplit) != 3 {
				return fmt.Errorf("unrecognized index line: '%s'", line)
			}
			source := strings.TrimSpace(lineSplit[0])
			eventType, ok := sensor.MatchEventType(strings.TrimSpace(lineSplit[1]))
			if !ok {
				return fmt.Errorf("unrecognized event type in line: '%s'", line)
			}
			match := ParseMatch(strings.TrimSpace(lineSplit[2]))
			i.addMatch(source, eventType, match...)
		}
		if eof {
			return nil
		}
	}
}

func (i *Index) Save(w io.Writer) (int, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return i.rootNode.Save(w)
}

func (i *Index) AddMatch(service string, eventType sensor.EventType, match ...Value) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.addMatch(service, eventType, match...)
}

func (i *Index) addMatch(service string, eventType sensor.EventType, match ...Value) {
	if len(match) == 0 {
		return
	}
	i.matchCount -= i.rootNode.AddMatch(service, eventType, match, 0, i.logger)
}

func (i *Index) Size() int {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return i.matchCount
}

func (i *Index) ResolveValues(s string) *ResolvedValues {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	resolver := &indexResolver{}
	tokens := i.tokenizer.Tokens(s)
	return resolver.ResolveValues(&i.rootNode, tokens, 0)
}

type indexNode struct {
	Value      Value
	valueNodes map[Value]*indexNode
	match      Match
	service    string
	eventType  sensor.EventType
}

func (n *indexNode) Save(w io.Writer) (int, error) {
	total := 0
	if n.match != nil {
		written, err := fmt.Fprintf(w, "%s:%s:%s\n", n.service, n.eventType, n.match.String())
		total += written
		if err != nil {
			return total, err
		}
	}
	for _, valueNode := range n.valueNodes {
		written, err := valueNode.Save(w)
		total += written
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (n *indexNode) AddMatch(service string, eventType sensor.EventType, match Match, valueIndex int, logger *slog.Logger) int {
	value := match[valueIndex]
	valueNode := n.valueNodes[value]
	if valueNode == nil {
		valueNode = &indexNode{
			Value: value,
		}
		if n.valueNodes == nil {
			n.valueNodes = make(map[Value]*indexNode)
		}
		n.valueNodes[value] = valueNode
	}
	nextValueIndex := valueIndex + 1
	final := nextValueIndex == len(match)
	added := 0
	if !final {
		added = valueNode.AddMatch(service, eventType, match, nextValueIndex, logger)
	} else {
		if valueNode.match == nil {
			added = 1
		} else {
			logger.Warn("replacing match", slog.Any("old", valueNode.match), slog.Any("new", match))
		}
		valueNode.match = make(Match, len(match))
		valueNode.eventType = eventType
		copy(valueNode.match, match)
	}
	return added
}

type indexResolver struct {
	match          Match
	service        string
	eventType      sensor.EventType
	matchingTokens []Token
}

func (r *indexResolver) resolve() *ResolvedValues {
	if r.match == nil {
		return nil
	}
	resolved := &ResolvedValues{
		Service:   r.service,
		EventType: r.eventType,
	}
	for index, value := range r.match {
		switch value {
		case HardwareAddressValue:
			resolved.HardwareAddress = r.matchingTokens[index].HardwareAddressValue()
		case IPAddressValue:
			resolved.IPAddress = r.matchingTokens[index].IPAddressValue()
		case UserValue:
			resolved.User = r.matchingTokens[index].Symbol
		case ServiceValue:
			resolved.Service = r.matchingTokens[index].Symbol
		}
	}
	return resolved
}

func (r *indexResolver) ResolveValues(node *indexNode, tokens []Token, tokenIndex int) *ResolvedValues {
	if node.match != nil {
		r.match = node.match
		r.service = node.service
		r.eventType = node.eventType
		r.matchingTokens = tokens[:tokenIndex]
	}
	if tokenIndex == len(tokens) {
		return r.resolve()
	}
	token := tokens[tokenIndex]
	if valueNode := node.valueNodes[Value(token.Symbol)]; valueNode != nil {
		return r.ResolveValues(valueNode, tokens, tokenIndex+1)
	}
	switch token.Type() {
	case TokenTypeHardwareAddress:
		if valueNode := node.valueNodes[HardwareAddressValue]; valueNode != nil {
			return r.ResolveValues(valueNode, tokens, tokenIndex+1)
		}
	case TokenTypeIPAddress:
		if valueNode := node.valueNodes[IPAddressValue]; valueNode != nil {
			return r.ResolveValues(valueNode, tokens, tokenIndex+1)
		}
	}
	if valueNode := node.valueNodes[UserValue]; valueNode != nil {
		return r.ResolveValues(valueNode, tokens, tokenIndex+1)
	} else if valueNode = node.valueNodes[ServiceValue]; valueNode != nil {
		return r.ResolveValues(valueNode, tokens, tokenIndex+1)
	} else if valueNode = node.valueNodes[AnyValue]; valueNode != nil {
		return r.ResolveValues(valueNode, tokens, tokenIndex+1)
	}
	return r.resolve()
}
