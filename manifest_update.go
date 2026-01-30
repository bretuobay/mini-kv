package minikv

import (
	"path/filepath"
	"strings"

	"mini-kv/internal/manifest"
	"mini-kv/internal/snapshot"
	"mini-kv/internal/wal"
)

func refreshManifest(dbPath string) error {
	walDir := filepath.Join(dbPath, "wal")
	snapDir := filepath.Join(dbPath, "snapshots")
	manifestPath := filepath.Join(dbPath, "MANIFEST")

	segments, err := wal.ListSegments(walDir)
	if err != nil {
		return err
	}
	walSegments := make([]manifest.WALSegment, 0, len(segments))
	var currentSeq uint64
	for _, path := range segments {
		seq, ok := parseSegmentSeq(path)
		if !ok {
			continue
		}
		if seq > currentSeq {
			currentSeq = seq
		}
		walSegments = append(walSegments, manifest.WALSegment{Seq: seq, Path: path})
	}

	snapshots, err := listSnapshots(snapDir)
	if err != nil {
		return err
	}
	snapInfos := make([]manifest.SnapshotInfo, 0, len(snapshots))
	var lastSnapSeq uint64
	for _, path := range snapshots {
		seq, ok := parseSnapshotSeq(path)
		if !ok {
			continue
		}
		if seq > lastSnapSeq {
			lastSnapSeq = seq
		}
		snapInfos = append(snapInfos, manifest.SnapshotInfo{Seq: seq, Path: path})
	}

	man := manifest.Manifest{
		CurrentWALSeq:   currentSeq,
		LastSnapshotSeq: lastSnapSeq,
		WALSegments:     walSegments,
		Snapshots:       snapInfos,
	}
	return manifest.WriteManifest(manifestPath, man)
}

func listSnapshots(dir string) ([]string, error) {
	mgr := snapshot.NewManager(dir)
	return mgr.ListSnapshots()
}

func parseSnapshotSeq(path string) (uint64, bool) {
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "snapshot_") || !strings.HasSuffix(base, ".snap") {
		return 0, false
	}
	value := strings.TrimSuffix(strings.TrimPrefix(base, "snapshot_"), ".snap")
	seq, err := parseUint(value)
	if err != nil {
		return 0, false
	}
	return seq, true
}
