package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

type manifestFixture struct {
	CurrentWALSeq   uint64
	LastSnapshotSeq uint64
	WALSegments     []WALSegment
	Snapshots       []SnapshotInfo
}

func TestManifestTracksState(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("manifest round trip", prop.ForAll(
		func(fixture manifestFixture) bool {
			dir := t.TempDir()
			path := filepath.Join(dir, "MANIFEST")

			manifest := Manifest{
				CurrentWALSeq:   fixture.CurrentWALSeq,
				LastSnapshotSeq: fixture.LastSnapshotSeq,
				WALSegments:     fixture.WALSegments,
				Snapshots:       fixture.Snapshots,
			}
			if err := WriteManifest(path, manifest); err != nil {
				return false
			}

			loaded, err := ReadManifest(path)
			if err != nil {
				return false
			}

			if loaded.CurrentWALSeq != manifest.CurrentWALSeq || loaded.LastSnapshotSeq != manifest.LastSnapshotSeq {
				return false
			}
			if !segmentsEqual(loaded.WALSegments, manifest.WALSegments) {
				return false
			}
			if !snapshotsEqual(loaded.Snapshots, manifest.Snapshots) {
				return false
			}
			return true
		},
		genManifestFixture(),
	))

	properties.TestingRun(t)
}

func TestManifestWriteIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MANIFEST")

	manifest := Manifest{CurrentWALSeq: 1}
	if err := WriteManifest(path, manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatalf("expected no tmp file after write")
	}
}

func genManifestFixture() gopter.Gen {
	return gopter.CombineGens(
		gen.UInt64(),
		gen.UInt64(),
		gen.SliceOf(genSegment()),
		gen.SliceOf(genSnapshot()),
	).Map(func(values []interface{}) manifestFixture {
		return manifestFixture{
			CurrentWALSeq:   values[0].(uint64),
			LastSnapshotSeq: values[1].(uint64),
			WALSegments:     values[2].([]WALSegment),
			Snapshots:       values[3].([]SnapshotInfo),
		}
	})
}

func genSegment() gopter.Gen {
	return gopter.CombineGens(
		gen.UInt64(),
		gen.AnyString(),
	).Map(func(values []interface{}) WALSegment {
		return WALSegment{Seq: values[0].(uint64), Path: values[1].(string)}
	})
}

func genSnapshot() gopter.Gen {
	return gopter.CombineGens(
		gen.UInt64(),
		gen.AnyString(),
	).Map(func(values []interface{}) SnapshotInfo {
		return SnapshotInfo{Seq: values[0].(uint64), Path: values[1].(string)}
	})
}

func segmentsEqual(a, b []WALSegment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func snapshotsEqual(a, b []SnapshotInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
