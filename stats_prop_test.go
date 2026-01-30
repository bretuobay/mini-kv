package minikv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestStatsProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("stats reflects database state", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			if err := db.Set([]byte(key), value); err != nil {
				return false
			}
			_, _ = db.Get([]byte(key))
			_, _ = db.Exists([]byte(key))
			_, _, _ = db.Scan([]byte(key), 10)

			stats, err := db.Stats()
			if err != nil {
				return false
			}
			return stats.KeyCount >= 1 && stats.Writes >= 1 && stats.Reads >= 2 && stats.Scans >= 1
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("dumpkeys writes all keys", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			if err := db.Set([]byte(key), value); err != nil {
				return false
			}

			var buf bytes.Buffer
			if err := db.DumpKeys(&buf); err != nil {
				return false
			}
			return strings.Contains(buf.String(), key+"\t")
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
