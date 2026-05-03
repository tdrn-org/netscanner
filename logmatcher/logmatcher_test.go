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

package logmatcher_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/logmatcher"
)

func TestMatchParse(t *testing.T) {
	match := logmatcher.Match([]logmatcher.Value{
		logmatcher.AnyValue,
		logmatcher.AddressValue,
		logmatcher.HardwareAddressValue,
		logmatcher.UserValue,
		logmatcher.ServiceValue,
		"no_quote",
		"{single_quote}",
		"{{double_quote}}",
	})
	matchString := match.String()
	require.Equal(t, "{Any} {IP} {MAC} {User} {Service} no_quote {{single_quote}} {{{{double_quote}}}}", matchString)
	parsed := logmatcher.ParseMatch(match.String())
	require.Equal(t, match, parsed)
}
