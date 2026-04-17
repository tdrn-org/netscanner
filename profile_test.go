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

package netscanner_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner"
)

func TestProfileSaveLoad(t *testing.T) {
	loadedProfile, err := netscanner.LoadProfileFile("testdata/test_profile.json")
	require.NoError(t, err)
	tempFile := filepath.Join(t.TempDir(), "test_profile.json")
	// Create case
	err = loadedProfile.SaveFile(tempFile)
	require.NoError(t, err)
	// Truncate case
	err = loadedProfile.SaveFile(tempFile)
	require.NoError(t, err)
	// Re-load
	reloadedProfile, err := netscanner.LoadProfileFile(tempFile)
	require.NoError(t, err)
	require.Equal(t, loadedProfile, reloadedProfile)
}
