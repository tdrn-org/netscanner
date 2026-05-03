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

package ouidb_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/netscanner/ouidb"
)

func TestPlainReader(t *testing.T) {
	db := ouidb.NewPlainReader(openTestOuiTxt(t))
	defer db.Close()

	vendorCount := 0
	for {
		vendor, err := db.ReadNext()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		fmt.Println(vendor.String())
		require.NotEmpty(t, vendor.Name)
		vendorCount++
	}
	require.Equal(t, 3, vendorCount)
}

func TestEmptyIndex(t *testing.T) {
	idxDir := t.TempDir()
	idxWriter, err := ouidb.NewIndexWriter(idxDir, t.Name())
	require.NoError(t, err)

	require.NoError(t, idxWriter.WriteIndex())
	require.NoError(t, idxWriter.Close())
}

func TestIndexWriterReader(t *testing.T) {
	db := ouidb.NewPlainReader(openTestOuiTxt(t))
	defer db.Close()

	idxDir := t.TempDir()
	idxWriter, err := ouidb.NewIndexWriter(idxDir, t.Name())
	require.NoError(t, err)

	for {
		vendor, err := db.ReadNext()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		err = idxWriter.WriteVendor(vendor)
		require.NoError(t, err)
	}

	err = idxWriter.WriteIndex()
	require.NoError(t, err)
	err = idxWriter.Close()
	require.NoError(t, err)

	idxReader, err := ouidb.NewIndexReader(idxDir, t.Name())
	require.NoError(t, err)
	defer idxReader.Close()

	// Miss 123456
	vendor123456ID := ouidb.VendorID([3]byte{0x12, 0x34, 0x56})
	vendor123456, err := idxReader.Lookup(vendor123456ID)
	require.NoError(t, err)
	require.Equal(t, ouidb.NoVendor, vendor123456)

	// Hit 002272
	vendor002272ID := ouidb.VendorID([3]byte{0x00, 0x22, 0x72})
	vendor002272, err := idxReader.Lookup(vendor002272ID)
	require.NoError(t, err)
	require.Equal(t, vendor002272ID, vendor002272.ID)

	// Hit 00D0EF
	vendor00D0EFID := ouidb.VendorID([3]byte{0x00, 0xd0, 0xef})
	vendor00D0EF, err := idxReader.Lookup(vendor00D0EFID)
	require.NoError(t, err)
	require.Equal(t, vendor00D0EFID, vendor00D0EF.ID)

	// Hit 086195
	vendor086195ID := ouidb.VendorID([3]byte{0x08, 0x61, 0x95})
	vendor086195, err := idxReader.Lookup(vendor086195ID)
	require.NoError(t, err)
	require.Equal(t, vendor086195ID, vendor086195.ID)
}

func TestDefaultIndexReader(t *testing.T) {
	idxReader := ouidb.DefaultIndexReader()
	require.NotNil(t, idxReader)

	// Miss fedcba
	vendorFEDCBAID := ouidb.VendorID([3]byte{0xfe, 0xdc, 0xba})
	vendorFEDCBA, err := idxReader.Lookup(vendorFEDCBAID)
	require.NoError(t, err)
	require.Equal(t, ouidb.NoVendor, vendorFEDCBA)

	// Hit 582BDB
	vendor582BDBID := ouidb.VendorID([3]byte{0x58, 0x2b, 0xdb})
	vendor582BDB, err := idxReader.Lookup(vendor582BDBID)
	require.NoError(t, err)
	require.Equal(t, vendor582BDBID, vendor582BDB.ID)
}

func TestLookupHardwareAddr(t *testing.T) {
	idxReader := ouidb.DefaultIndexReader()
	require.NotNil(t, idxReader)

	// Miss fedcba
	hardwareAddrFEDCBA, err := net.ParseMAC("FE:DC:BA:01:02:03")
	require.NoError(t, err)
	vendorFEDCBA, err := idxReader.LookupHardwareAddr(hardwareAddrFEDCBA)
	require.NoError(t, err)
	require.Equal(t, ouidb.NoVendor, vendorFEDCBA)

	// Hit 582BDB
	vendor582BDBID := ouidb.VendorID([3]byte{0x58, 0x2b, 0xdb})
	hardwareAddr582BDB, err := net.ParseMAC("58:2b:db:01:02:03")
	require.NoError(t, err)
	vendor582BDB, err := idxReader.LookupHardwareAddr(hardwareAddr582BDB)
	require.NoError(t, err)
	require.Equal(t, vendor582BDBID, vendor582BDB.ID)
}

func openTestOuiTxt(t *testing.T) *os.File {
	file, err := os.Open("testdata/oui.txt")
	require.NoError(t, err)
	return file
}
