package wal

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ReadWAL reads and decodes WAL records from a segment path.
// Stops at the first corrupt record and returns the records read so far.
func ReadWAL(path string) ([]WALRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	records := make([]WALRecord, 0)
	off := 0
	for off < len(data) {
		rec, consumed, err := DecodeWALRecord(data[off:])
		if err != nil {
			return records, nil
		}
		if consumed == 0 {
			return records, nil
		}
		records = append(records, rec)
		off += consumed
	}

	return records, nil
}

// ListSegments returns WAL segment paths in increasing sequence order.
func ListSegments(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type segment struct {
		seq  uint64
		path string
	}
	segments := make([]segment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}
		base := strings.TrimSuffix(name, ".log")
		seq, err := strconv.ParseUint(base, 10, 64)
		if err != nil {
			continue
		}
		segments = append(segments, segment{seq: seq, path: filepath.Join(dir, name)})
	}

	sort.Slice(segments, func(i, j int) bool { return segments[i].seq < segments[j].seq })
	paths := make([]string, 0, len(segments))
	for _, s := range segments {
		paths = append(paths, s.path)
	}
	return paths, nil
}
