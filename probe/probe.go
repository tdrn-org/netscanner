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
	"context"
	"io"
	"net/netip"
)

type Result interface {
	Address() netip.Addr
	Error() error
	String() string
	Up() bool
}

type Runner[R Result] interface {
	Run(ctx context.Context, address netip.Addr) R
}

func formatResult(buffer io.StringWriter, r Result) {
	buffer.WriteString(r.Address().String())
	if r.Up() {
		buffer.WriteString(" up")
	} else {
		buffer.WriteString(" down")
	}
}
