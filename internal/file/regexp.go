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

package file

import (
	"bytes"
	"io"
	"regexp"
)

type RegexpDecoder struct {
	Pattern     *regexp.Regexp
	buffer      []byte
	pos         int
	matchBuffer []byte
	decoded     []string
}

func (d *RegexpDecoder) Nil() []string {
	return nil
}

func (d *RegexpDecoder) Feed(r io.Reader) (int, error) {
	if d.decode() {
		return 0, nil
	}
	if (len(d.buffer) - d.pos) < minFeedSize {
		buffer := make([]byte, len(d.buffer)+defaultFeedSize)
		copy(buffer, d.buffer)
		d.buffer = buffer
	}
	read, err := r.Read(d.buffer[d.pos:])
	d.pos += read
	return read, err
}

func (d *RegexpDecoder) Decode() ([]string, error) {
	if !d.decode() {
		return nil, nil
	}
	decoded := d.decoded
	d.decoded = nil
	return decoded, nil
}

func (d *RegexpDecoder) decode() bool {
	if d.decoded != nil {
		return true
	}
	index := bytes.IndexByte(d.buffer[:d.pos], '\n')
	if index < 0 {
		return false
	}
	lineLen := index + 1
	d.matchBuffer = append(d.matchBuffer, d.buffer[0:lineLen]...)
	copy(d.buffer, d.buffer[lineLen:])
	d.pos -= lineLen
	decoded := d.Pattern.FindStringSubmatch(string(d.matchBuffer))
	if decoded == nil {
		return false
	}
	d.matchBuffer = d.matchBuffer[:0]
	d.decoded = decoded
	return true
}
