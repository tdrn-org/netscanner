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
	"net"
	"net/netip"
	"strings"
)

type TokenType int

const (
	TokenTypeUnknown         TokenType = 0
	TokenTypeHardwareAddress TokenType = 1
	TokenTypeIPAddress       TokenType = 2
	TokenTypeSymbol          TokenType = 3
)

type Token struct {
	Symbol   string
	typeHint TokenType
	value    any
}

func (t *Token) resolve() (TokenType, any) {
	if t.typeHint != TokenTypeUnknown {
		return t.typeHint, t.value
	}
	hardwareAddress, err := net.ParseMAC(t.Symbol)
	if err == nil {
		t.typeHint = TokenTypeHardwareAddress
		t.value = hardwareAddress
		return t.typeHint, t.value
	}
	ipAddress, err := netip.ParseAddr(t.Symbol)
	if err == nil {
		t.typeHint = TokenTypeIPAddress
		t.value = &ipAddress
		return t.typeHint, t.value
	}
	t.typeHint = TokenTypeSymbol
	t.value = t.Symbol
	return t.typeHint, t.value
}

func (t *Token) Type() TokenType {
	typeHint, _ := t.resolve()
	return typeHint
}

func (t *Token) Value() any {
	_, value := t.resolve()
	return value
}

func (t *Token) HardwareAddressValue() net.HardwareAddr {
	typeHint, value := t.resolve()
	if typeHint != TokenTypeHardwareAddress {
		return nil
	}
	return value.(net.HardwareAddr)
}

func (t *Token) IPAddressValue() *netip.Addr {
	typeHint, value := t.resolve()
	if typeHint != TokenTypeIPAddress {
		return nil
	}
	return value.(*netip.Addr)
}

type Tokenizer interface {
	Tokens(s string) []Token
}

type TokenizerFunc func(s string) []Token

func (t TokenizerFunc) Tokens(s string) []Token {
	return t(s)
}

var FieldsTokenizer TokenizerFunc = func(s string) []Token {
	symbols := strings.Fields(s)
	tokens := make([]Token, 0, len(symbols))
	for _, symbol := range symbols {
		tokens = append(tokens, Token{Symbol: symbol})
	}
	return tokens
}
