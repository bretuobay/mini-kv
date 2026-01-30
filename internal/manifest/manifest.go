package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WALSegment describes a WAL log segment.
type WALSegment struct {
	Seq  uint64
	Path string
}

// SnapshotInfo describes a snapshot file.
type SnapshotInfo struct {
	Seq  uint64
	Path string
}

// Manifest tracks WAL and snapshot state.
type Manifest struct {
	CurrentWALSeq   uint64
	LastSnapshotSeq uint64
	WALSegments     []WALSegment
	Snapshots       []SnapshotInfo
}

// ReadManifest loads a manifest from disk.
func ReadManifest(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer file.Close()

	manifest := Manifest{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return Manifest{}, fmt.Errorf("manifest: invalid line %q", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "current_wal_seq":
			manifest.CurrentWALSeq, err = parseUint(value)
		case "last_snapshot_seq":
			manifest.LastSnapshotSeq, err = parseUint(value)
		case "wal":
			seg, parseErr := parseSegment(value)
			if parseErr != nil {
				return Manifest{}, parseErr
			}
			manifest.WALSegments = append(manifest.WALSegments, seg)
		case "snapshot":
			snap, parseErr := parseSnapshot(value)
			if parseErr != nil {
				return Manifest{}, parseErr
			}
			manifest.Snapshots = append(manifest.Snapshots, snap)
		}
		if err != nil {
			return Manifest{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

// WriteManifest writes the manifest atomically using a temp file + rename.
func WriteManifest(path string, manifest Manifest) error {
	tmpPath := path + ".tmp"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	if _, err := fmt.Fprintf(writer, "current_wal_seq: %d\n", manifest.CurrentWALSeq); err != nil {
		file.Close()
		return err
	}
	if _, err := fmt.Fprintf(writer, "last_snapshot_seq: %d\n", manifest.LastSnapshotSeq); err != nil {
		file.Close()
		return err
	}
		for _, seg := range manifest.WALSegments {
			path := strconv.Quote(seg.Path)
			if _, err := fmt.Fprintf(writer, "wal: %d %s\n", seg.Seq, path); err != nil {
				file.Close()
				return err
			}
		}
		for _, snap := range manifest.Snapshots {
			path := strconv.Quote(snap.Path)
			if _, err := fmt.Fprintf(writer, "snapshot: %d %s\n", snap.Seq, path); err != nil {
				file.Close()
				return err
			}
		}
	if err := writer.Flush(); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func parseUint(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}

func parseSegment(value string) (WALSegment, error) {
	seqStr, pathStr, ok := splitSeqPath(value)
	if !ok {
		return WALSegment{}, fmt.Errorf("manifest: invalid wal segment %q", value)
	}
	seq, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		return WALSegment{}, err
	}
	path, err := strconv.Unquote(pathStr)
	if err != nil {
		return WALSegment{}, err
	}
	return WALSegment{Seq: seq, Path: path}, nil
}

func parseSnapshot(value string) (SnapshotInfo, error) {
	seqStr, pathStr, ok := splitSeqPath(value)
	if !ok {
		return SnapshotInfo{}, fmt.Errorf("manifest: invalid snapshot %q", value)
	}
	seq, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		return SnapshotInfo{}, err
	}
	path, err := strconv.Unquote(pathStr)
	if err != nil {
		return SnapshotInfo{}, err
	}
	return SnapshotInfo{Seq: seq, Path: path}, nil
}

func splitSeqPath(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	idx := strings.IndexByte(value, ' ')
	if idx <= 0 {
		return "", "", false
	}
	seqStr := strings.TrimSpace(value[:idx])
	pathStr := strings.TrimSpace(value[idx+1:])
	if seqStr == "" || pathStr == "" {
		return "", "", false
	}
	return seqStr, pathStr, true
}
