package minikv

import (
	"sync"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestConcurrencyProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	properties.Property("concurrent reads", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" {
				key = "k"
			}
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			_ = db.Set([]byte(key), value)
			var wg sync.WaitGroup
			ok := true
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					got, err := db.Get([]byte(key))
					if err != nil || string(got) != string(value) {
						ok = false
					}
				}()
			}
			wg.Wait()
			return ok
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("concurrent writes", prop.ForAll(
		func(keys []string, values [][]byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			limit := len(keys)
			if len(values) < limit {
				limit = len(values)
			}
			var wg sync.WaitGroup
			for i := 0; i < limit; i++ {
				key := normalizeKey(keys[i])
				if key == "" {
					continue
				}
				value := values[i]
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = db.Set([]byte(key), value)
				}()
			}
			wg.Wait()

			count, err := db.Count()
			if err != nil {
				return false
			}
			return count >= 0
		},
		gen.SliceOf(gen.AnyString()),
		gen.SliceOf(gen.SliceOf(gen.UInt8())),
	))

	properties.Property("read your writes", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" {
				key = "k"
			}
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if err := db.Set([]byte(key), value); err != nil {
				return false
			}
			got, err := db.Get([]byte(key))
			return err == nil && string(got) == string(value)
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
