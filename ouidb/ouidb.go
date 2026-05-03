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

package ouidb

//go:generate go run -tags=tools ../cmd/build/build.go ouidb https://www.linuxnet.ca/oui/oui.txt

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type VendorID [3]byte

func (id VendorID) Equal(vendorID VendorID) bool {
	return id[0] == vendorID[0] && id[1] == vendorID[1] && id[2] == vendorID[2]
}

func (id VendorID) Compare(vendorID VendorID) int {
	return bytes.Compare(id[:], vendorID[:])
}

type Vendor struct {
	ID   VendorID
	Name string
}

func (v *Vendor) String() string {
	return fmt.Sprintf("%02x:%02x:%02x '%s'", v.ID[0], v.ID[1], v.ID[2], v.Name)
}

var NoVendor Vendor = Vendor{}

type PlainReader struct {
	in     *bufio.Reader
	closer io.Closer
}

func NewPlainReader(r io.ReadCloser) *PlainReader {
	return &PlainReader{
		in:     bufio.NewReader(r),
		closer: r,
	}
}

var ouiTxtLinePattern *regexp.Regexp = regexp.MustCompile(`^([[:xdigit:]]{2})-([[:xdigit:]]{2})-([[:xdigit:]]{2})[[:space:]]+\(hex\)[[:space:]]+(.+)$`)

func (r *PlainReader) ReadNext() (Vendor, error) {
	for {
		line, err := r.in.ReadString('\n')
		eof := errors.Is(err, io.EOF)
		if err != nil && !eof {
			return NoVendor, err
		}
		line = strings.TrimSpace(line)
		match := ouiTxtLinePattern.FindStringSubmatch(line)
		if match != nil {
			id0, id0Err := strconv.ParseUint(match[1], 16, 8)
			id1, id1Err := strconv.ParseUint(match[2], 16, 8)
			id2, id2Err := strconv.ParseUint(match[3], 16, 8)
			err := errors.Join(id0Err, id1Err, id2Err)
			name := match[4]
			if name == "" {
				err = errors.Join(err, errors.New("empty vendor name"))
			}
			if err != nil {
				return NoVendor, fmt.Errorf("unrecognized vendor definition '%s' (cause: %w)", match[0], err)
			}
			vendor := Vendor{
				ID:   [3]byte{byte(id0), byte(id1), byte(id2)},
				Name: name,
			}
			return vendor, nil
		}
		if eof {
			return NoVendor, err
		}
	}

}

func (r *PlainReader) Close() error {
	return r.closer.Close()
}

type IndexWriter struct {
	datName string
	datFile *os.File
	datPos  int64
	idxName string
	entries [][8]byte
}

func NewIndexWriter(dir, name string) (*IndexWriter, error) {
	datName := filepath.Join(dir, name+".dat")
	datFile, err := os.Create(datName)
	if err != nil {
		return nil, err
	}
	idxName := filepath.Join(dir, name+".idx")
	indexWriter := &IndexWriter{
		datName: datName,
		datFile: datFile,
		idxName: idxName,
	}
	return indexWriter, nil
}

func (w *IndexWriter) WriteVendor(vendor Vendor) error {
	nameBytes := []byte(vendor.Name)
	nameLen := int64(len(nameBytes))
	if nameLen > 0xff {
		return fmt.Errorf("vendor name '%s' exceeds lenght limit", vendor.Name)
	}
	if w.datPos+nameLen > int64(math.MaxUint32) {
		return fmt.Errorf("not enough index space to write vendor %s", vendor.String())
	}
	entry := [8]byte(make([]byte, 8))
	entry[0] = vendor.ID[0]
	entry[1] = vendor.ID[1]
	entry[2] = vendor.ID[2]
	entry[3] = byte(nameLen)
	binary.LittleEndian.PutUint32(entry[4:8], uint32(w.datPos))
	_, err := w.datFile.Write(nameBytes)
	if err != nil {
		return err
	}
	w.datPos += int64(nameLen)
	w.entries = append(w.entries, entry)
	return nil
}

func (w *IndexWriter) WriteIndex() error {
	slices.SortFunc(w.entries, func(e1, e2 [8]byte) int {
		return bytes.Compare(e1[:3], e2[:3])
	})
	idxFile, err := os.Create(w.idxName)
	if err != nil {
		return err
	}
	defer idxFile.Close()
	for _, entry := range w.entries {
		_, err = idxFile.Write(entry[:])
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *IndexWriter) Close() error {
	return w.datFile.Close()
}

type IndexReader struct {
	idx             []byte
	datFileReaderAt io.ReaderAt
	datFileCloser   io.Closer
}

func NewIndexReader(dir, name string) (*IndexReader, error) {
	idxName := filepath.Join(dir, name+".idx")
	idxBytes, err := os.ReadFile(idxName)
	if err != nil {
		return nil, err
	}
	datName := filepath.Join(dir, name+".dat")
	datFile, err := os.Open(datName)
	if err != nil {
		return nil, err
	}
	indexReader, err := NewEmbedIndexReader(idxBytes, datFile)
	if err != nil {
		return nil, errors.Join(err, datFile.Close())
	}
	return indexReader, nil
}

func NewEmbedIndexReader(idx []byte, datFile io.ReadCloser) (*IndexReader, error) {
	idxLen := len(idx)
	if idxLen%8 != 0 {
		return nil, fmt.Errorf("invalid index data (len: %d)", idxLen)
	}
	datFileReaderAt, ok := datFile.(io.ReaderAt)
	if !ok {
		return nil, fmt.Errorf("incompatible index reader: %s", reflect.TypeOf(datFile).String())
	}
	indexReader := &IndexReader{
		idx:             idx,
		datFileReaderAt: datFileReaderAt,
		datFileCloser:   datFile,
	}
	return indexReader, nil
}

func (r *IndexReader) LookupHardwareAddr(address net.HardwareAddr) (Vendor, error) {
	return r.Lookup(VendorID(address[0:3]))
}

func (r *IndexReader) Lookup(id VendorID) (Vendor, error) {
	n := len(r.idx)
	i, j := 0, n
	for i < j {
		h := int((uint(i+j) >> 4) << 3)
		if bytes.Compare(r.idx[h:h+3], id[:]) < 0 {
			i = h + 8
		} else {
			j = h
		}
	}
	if !(i < n && bytes.Equal(r.idx[i:i+3], id[:])) {
		return NoVendor, nil
	}
	nameLen := int(uint(r.idx[i+3]))
	nameOff := int64(uint64(binary.LittleEndian.Uint32(r.idx[i+4 : i+8])))
	name := make([]byte, nameLen)
	_, err := r.datFileReaderAt.ReadAt(name, nameOff)
	if err != nil {
		return NoVendor, err
	}
	vendor := Vendor{
		ID:   id,
		Name: string(name),
	}
	return vendor, nil
}

func (r *IndexReader) Close() error {
	return r.datFileCloser.Close()
}

var defaultIndexReader *IndexReader

func DefaultIndexReader() *IndexReader {
	return defaultIndexReader
}

//go:embed ouidb.idx
var ouidbIdx []byte

//go:embed ouidb.dat
var ouidbDatFS embed.FS

func init() {
	ouidbDatFile, err := ouidbDatFS.Open("ouidb.dat")
	if err != nil {
		panic(err)
	}
	indexReader, err := NewEmbedIndexReader(ouidbIdx, ouidbDatFile)
	if err != nil {
		panic(err)
	}
	defaultIndexReader = indexReader
}
