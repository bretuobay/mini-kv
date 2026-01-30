package minikv

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestReadOnlyAndClosedProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	properties.Property("read-only mode rejects writes", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" {
				key = "k"
			}
			opts := DefaultOptions(t.TempDir())
			opts.ReadOnly = true
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()

			if err := db.Set([]byte(key), value); err != ErrReadOnly {
				return false
			}
			if err := db.Delete([]byte(key)); err != ErrReadOnly {
				return false
			}
			_, err = db.Incr([]byte(key))
			return err == ErrReadOnly
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("closed database returns ErrClosed", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" {
				key = "k"
			}
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			if err := db.Close(); err != nil {
				return false
			}
			if _, err := db.Get([]byte(key)); err != ErrClosed {
				return false
			}
			if err := db.Set([]byte(key), value); err != ErrClosed {
				return false
			}
			return true
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
