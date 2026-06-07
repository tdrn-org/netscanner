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
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	framestream "github.com/farsightsec/golang-framestream"
	"google.golang.org/protobuf/proto"
)

type fileReceiver struct {
	path         string
	file         *os.File
	frameReader  *framestream.Reader
	delay        time.Duration
	maxFrameSize int
	skipBefore   time.Time
	logger       *slog.Logger
	stopping     atomic.Bool
	stoppedWG    sync.WaitGroup
}

func NewFileReceiver(path string, maxFrameSize int, tail bool) (Receiver, error) {
	skipBefore := time.Unix(0, 0)
	if tail {
		skipBefore = time.Now()
	}
	receiver := &fileReceiver{
		path:         path,
		maxFrameSize: maxFrameSize,
		skipBefore:   skipBefore,
		logger:       slog.With(slog.String("file", path)),
	}
	return receiver, nil
}

func (r *fileReceiver) Consume(consumer EntryConsumer) {
	r.stoppedWG.Add(1)
	defer r.stoppedWG.Done()

	buffer := make([]byte, r.maxFrameSize)
	for {
		if r.stopping.Load() {
			r.close()
			return
		}
		if r.delay != 0 {
			time.Sleep(r.delay)
			r.delay = 0
		}
		if !r.ensureOpen() {
			continue
		}
		frameLen, err := r.frameReader.ReadFrame(buffer)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			if !r.seekIfTruncated() {
				r.delay = fileReadDelay
			}
			continue
		} else if err != nil {
			r.logger.Error("file read failure", slog.Any("err", err))
			r.triggerReopen(fileReadDelay)
			continue
		}
		content := &Dnstap{}
		err = proto.Unmarshal(buffer[:frameLen], content)
		if err != nil {
			continue
		}
		entry := &Entry{
			Content:    content,
			skipBefore: r.skipBefore,
		}
		r.logger.Debug("consuming dnstap event")
		consumer(entry)
	}
}

const fileReadDelay time.Duration = 500 * time.Millisecond

func (r *fileReceiver) ensureOpen() bool {
	if r.frameReader == nil {
		r.logger.Info("opening dnstap...")
		file, err := os.Open(r.path)
		if err != nil {
			r.logger.Info("file not accessible; retrying after delay", slog.Any("cause", err))
			r.triggerReopen(2 * fileReadDelay)
			return false
		}
		r.file = file
		if !r.skipBefore.IsZero() {
			r.skipBefore = time.Now()
		}
		frameReader, err := framestream.NewReader(&followReader{r: r}, &framestream.ReaderOptions{
			ContentTypes:  [][]byte{[]byte("protobuf:dnstap.Dnstap")},
			Bidirectional: false,
		})
		if err != nil {
			r.logger.Info("failed to create frame reader; retrying after delay", slog.Any("cause", err))
			file.Close()
			r.triggerReopen(2 * fileReadDelay)
			return false
		}
		r.frameReader = frameReader
	}
	return true
}

type followReader struct {
	r *fileReceiver
}

func (f *followReader) Read(p []byte) (int, error) {
	return f.r.readFile(p)
}

func (r *fileReceiver) readFile(buffer []byte) (int, error) {
	for {
		if r.stopping.Load() {
			return 0, io.EOF
		}
		n, err := r.file.Read(buffer)
		if n > 0 {
			return n, err
		}
		if errors.Is(err, io.EOF) {
			if r.seekIfTruncated() {
				return 0, io.EOF
			}
			time.Sleep(fileReadDelay)
			continue
		}
		return n, err
	}
}

func (r *fileReceiver) triggerReopen(delay time.Duration) {
	r.close()
	r.delay = delay
}

func (r *fileReceiver) seekIfTruncated() bool {
	info, err := r.file.Stat()
	if err != nil {
		r.logger.Info("file stat failure; reopening file", slog.Any("cause", err))
		r.triggerReopen(0)
		return true
	}
	off, err := r.file.Seek(0, io.SeekCurrent)
	if err != nil {
		r.logger.Info("file seek current failure; reopening file", slog.Any("cause", err))
		r.triggerReopen(0)
		return true
	}
	if off <= info.Size() {
		return false
	}
	r.logger.Info("file truncated; seeking to start")
	_, err = r.file.Seek(0, io.SeekStart)
	if err != nil {
		r.logger.Info("file seek start failure; reopening file", slog.Any("cause", err))
		r.triggerReopen(0)
	}
	return true
}

func (r *fileReceiver) close() {
	if r.file != nil {
		r.logger.Info("closing file...")
		err := r.file.Close()
		if err != nil {
			r.logger.Warn("close file failure", slog.Any("err", err))
		}
		r.file = nil
		r.frameReader = nil
	}
}

func (r *fileReceiver) Shutdown(_ context.Context) error {
	return r.Close()
}

func (r *fileReceiver) Close() error {
	r.stopping.Store(true)
	r.stoppedWG.Wait()
	return nil
}
