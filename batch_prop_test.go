package minikv

import (
	"bytes"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestBatchProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("batch set applies all operations", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			if len(keys) == 0 || len(values) == 0 {
				return true
			}
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			batch := db.NewBatch()
			limit := len(keys)
			if len(values) < limit {
				limit = len(values)
			}
			expected := make(map[string][]byte, limit)
			for i := 0; i < limit; i++ {
				key := keys[i]
				if key == "" {
					key = "k"
				}
				keyBytes := []byte(key)
				if len(keyBytes) > MaxKeySize {
					keyBytes = keyBytes[:MaxKeySize]
					key = string(keyBytes)
				}
				value := values[i]
				if len(value) > MaxValueSize {
					value = value[:MaxValueSize]
				}
				batch.Set(keyBytes, value)
				expected[key] = append([]byte(nil), value...)
			}
			if err := batch.Write(); err != nil {
				return false
			}

			for key, value := range expected {
				got, err := db.Get([]byte(key))
				if err != nil {
					return false
				}
				if !bytes.Equal(got, value) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("batch respects ttl", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()
			if key == "" {
				key = "k"
			}

			batch := db.NewBatch()
			batch.SetWithTTL([]byte(key), value, 5*time.Millisecond)
			if err := batch.Write(); err != nil {
				return false
			}
			if _, err := db.Get([]byte(key)); err != nil {
				return false
			}
			time.Sleep(10 * time.Millisecond)
			_, err = db.Get([]byte(key))
			return err == ErrNotFound
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("batch operations buffered until write", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			batch := db.NewBatch()
			batch.Set([]byte(key), value)
			if _, err := db.Get([]byte(key)); err != ErrNotFound {
				return false
			}
			if err := batch.Write(); err != nil {
				return false
			}
			_, err = db.Get([]byte(key))
			return err == nil
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("batch discard abandons operations", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			batch := db.NewBatch()
			batch.Set([]byte(key), value)
			batch.Discard()
			if err := batch.Write(); err != ErrClosed {
				return false
			}
			_, err = db.Get([]byte(key))
			return err == ErrNotFound
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("batch size validation", prop.ForAll(
		func(value []byte) bool {
			opts := DefaultOptions(t.TempDir())
			opts.MaxBatchSize = 1
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()

			batch := db.NewBatch()
			batch.Set([]byte("k"), append(value, 0x1))
			return batch.Write() == ErrBatchTooBig
		},
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
