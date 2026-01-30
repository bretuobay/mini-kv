package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// WALManager handles append-only WAL files and rotation.
type WALManager struct {
	mu          sync.Mutex
	dir         string
	currentFile *os.File
	currentSeq  uint64
	currentSize int64
	maxSize     int64
	rotateHook  func()
}

// OpenWAL creates or opens a WAL directory and prepares the current segment.
func OpenWAL(dir string, maxSize int64) (*WALManager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	seq, err := latestSequence(dir)
	if err != nil {
		return nil, err
	}
	if seq == 0 {
		seq = 1
	}
	file, size, err := openSegment(dir, seq)
	if err != nil {
		return nil, err
	}
	return &WALManager{
		dir:         dir,
		currentFile: file,
		currentSeq:  seq,
		currentSize: size,
		maxSize:     maxSize,
	}, nil
}

// AppendRecord writes an encoded record to the WAL, rotating if needed.
func (w *WALManager) AppendRecord(record WALRecord) error {
	encoded := EncodeWALRecord(record)
	_, err := w.AppendRaw(encoded)
	return err
}

// AppendRaw writes a pre-encoded record to the WAL, rotating if needed.
func (w *WALManager) AppendRaw(encoded []byte) (int, error) {

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile == nil {
		return 0, os.ErrInvalid
	}

	if w.maxSize > 0 && w.currentSize+int64(len(encoded)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.currentFile.Write(encoded)
	if err != nil {
		return 0, err
	}
	w.currentSize += int64(n)
	return n, nil
}

// Sync flushes WAL data to disk.
func (w *WALManager) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.currentFile == nil {
		return os.ErrInvalid
	}
	return w.currentFile.Sync()
}

// Close closes the WAL file handle.
func (w *WALManager) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.currentFile == nil {
		return nil
	}
	err := w.currentFile.Close()
	w.currentFile = nil
	return err
}

// CurrentSeq returns the current WAL segment sequence.
func (w *WALManager) CurrentSeq() uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentSeq
}

func (w *WALManager) rotate() error {
	if w.currentFile != nil {
		if err := w.currentFile.Sync(); err != nil {
			return err
		}
		if err := w.currentFile.Close(); err != nil {
			return err
		}
	}
	w.currentSeq++
	file, size, err := openSegment(w.dir, w.currentSeq)
	if err != nil {
		return err
	}
	w.currentFile = file
	w.currentSize = size
	if w.rotateHook != nil {
		w.rotateHook()
	}
	return nil
}

// SetRotateHook registers a callback invoked after WAL rotation.
func (w *WALManager) SetRotateHook(hook func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rotateHook = hook
}

func openSegment(dir string, seq uint64) (*os.File, int64, error) {
	path := filepath.Join(dir, segmentName(seq))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

func latestSequence(dir string) (uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	seqs := make([]uint64, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}
		base := strings.TrimSuffix(name, ".log")
		value, err := strconv.ParseUint(base, 10, 64)
		if err != nil {
			continue
		}
		seqs = append(seqs, value)
	}
	if len(seqs) == 0 {
		return 0, nil
	}
	sort.Slice(seqs, func(i, j int) bool { return seqs[i] < seqs[j] })
	return seqs[len(seqs)-1], nil
}

func segmentName(seq uint64) string {
	return fmt.Sprintf("%06d.log", seq)
}
