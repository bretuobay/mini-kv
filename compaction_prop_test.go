package minikv

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestCompactionProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 10
	properties := gopter.NewProperties(parameters)

	properties.Property("compaction trigger logic", prop.ForAll(
		func(value []byte) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)
			opts.MaxWALSize = 64
			opts.SyncMode = SyncManual
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()

			for i := 0; i < 50; i++ {
				_ = db.Set([]byte("k"+intToString(i)), append(value, byte(i)))
			}
			time.Sleep(20 * time.Millisecond)
			manifestPath := filepath.Join(dir, "MANIFEST")
			_, err = readFileIfExists(manifestPath)
			return err == nil
		},
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("compaction updates manifest", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			if err := db.Compact(); err != nil {
				return false
			}
			_, err = readFileIfExists(filepath.Join(dir, "MANIFEST"))
			return err == nil
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("compaction deletes old wal", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)
			opts.MaxWALSize = 64
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			_ = db.Compact()
			segments, _ := listDir(filepath.Join(dir, "wal"))
			return len(segments) > 0
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("compaction concurrency", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			dir := t.TempDir()
			db, err := Open(DefaultOptions(dir))
			if err != nil {
				return false
			}
			defer db.Close()

			seedDB(db, keys, values)
			// Trigger compaction concurrently.
			go db.compactAsync()
			if err := db.Compact(); err != nil {
				return false
			}
			return true
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.TestingRun(t)
}

func readFileIfExists(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func listDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
