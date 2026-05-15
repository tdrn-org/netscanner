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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidJSONPath error = errors.New("invalid JSON path")
var ErrUnknownJSONPath error = fmt.Errorf("%w; not found", ErrInvalidJSONPath)
var ErrEmptyJSONPath error = fmt.Errorf("%w; empty path", ErrUnknownJSONPath)
var ErrJSONTypeMismatch error = errors.New("JSON type mismatch")

type JSON map[string]any

type JSONPath []string

func (p JSONPath) Len() int {
	return len(p)
}

func (p JSONPath) String() string {
	return strings.Join(p, "/")
}

func JSONValue[T any](object JSON, names ...string) (T, error) {
	var defaultValue T
	current := object
	for i, name := range names {
		if i == len(names)-1 {
			value, ok := current[name].(T)
			if !ok {
				return defaultValue, ErrJSONTypeMismatch
			}
			return value, nil
		}
		next, ok := current[name].(map[string]any)
		if !ok {
			return defaultValue, ErrInvalidJSONPath
		}
		current = next
	}
	return defaultValue, ErrEmptyJSONPath
}

func JSONValueToString(object JSON, names ...string) (string, error) {
	encoded, err := JSONValue[any](object, names...)
	if err != nil {
		return "", err
	}
	switch encoded := encoded.(type) {
	case string:
		return encoded, nil
	default:
		return "", fmt.Errorf("%w; unable to convert type %s to string", ErrJSONTypeMismatch, reflect.TypeOf(encoded).Name())
	}
}

func JSONValueToTime(layout string, object JSON, names ...string) (time.Time, error) {
	encoded, err := JSONValue[any](object, names...)
	if err != nil {
		return time.Time{}, err
	}
	switch encoded := encoded.(type) {
	case float64:
		sec := int64(encoded)
		nsec := int64(math.Round((encoded - float64(sec)) * 1000000000.0))
		decoded := time.Unix(sec, nsec)
		return decoded, nil
	case string:
		decoded, err := time.Parse(layout, encoded)
		if err != nil {
			return time.Time{}, fmt.Errorf("%w; failed to parse time value '%s'", ErrJSONTypeMismatch, encoded)
		}
		return decoded, nil
	default:
		return time.Time{}, fmt.Errorf("%w; unable to convert type %s to time", ErrJSONTypeMismatch, reflect.TypeOf(encoded).Name())
	}
}

func JSONValueToInt(object JSON, names ...string) (int, error) {
	encoded, err := JSONValue[any](object, names...)
	if err != nil {
		return 0, err
	}
	switch encoded := encoded.(type) {
	case float64:
		decoded := int(encoded)
		return decoded, nil
	case string:
		decoded, err := strconv.ParseInt(encoded, 10, bits.UintSize)
		if err != nil {
			return 0, fmt.Errorf("%w; failed to parse int value '%s'", ErrJSONTypeMismatch, encoded)
		}
		return int(decoded), nil
	default:
		return 0, fmt.Errorf("%w; unable to convert type %s to int", ErrJSONTypeMismatch, reflect.TypeOf(encoded).Name())
	}
}

type JSONDecoder struct {
	buffer  []byte
	pos     int
	decoded JSON
}

func (d *JSONDecoder) Nil() JSON {
	return nil
}

func (d *JSONDecoder) Feed(r io.Reader) (int, error) {
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

func (d *JSONDecoder) Decode() (JSON, error) {
	if !d.decode() {
		return nil, nil
	}
	decoded := d.decoded
	d.decoded = nil
	return decoded, nil
}

func (d *JSONDecoder) decode() bool {
	if d.decoded != nil {
		return true
	}
	reader := bytes.NewReader(d.buffer[:d.pos])
	decoder := json.NewDecoder(reader)
	var decoded JSON
	err := decoder.Decode(&decoded)
	if err != nil {
		return false
	}
	decodedLen := int(decoder.InputOffset())
	d.decoded = decoded
	copy(d.buffer, d.buffer[decodedLen:])
	d.pos -= decodedLen
	return true
}
