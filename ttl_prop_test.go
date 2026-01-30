package minikv

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestTTLProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("ttl expiration", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			if err := db.SetWithTTL([]byte(key), value, 5*time.Millisecond); err != nil {
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

	properties.Property("ttl remaining duration", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			if err := db.SetWithTTL([]byte(key), value, 50*time.Millisecond); err != nil {
				return false
			}
			remaining, err := db.TTL([]byte(key))
			if err != nil {
				return false
			}
			return remaining > 0
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("expire sets expiration", prop.ForAll(
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
			ok, err := db.Expire([]byte(key), 20*time.Millisecond)
			if err != nil || !ok {
				return false
			}
			remaining, err := db.TTL([]byte(key))
			if err != nil {
				return false
			}
			return remaining > 0
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("persist removes expiration", prop.ForAll(
		func(key string, value []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			if key == "" {
				key = "k"
			}
			if err := db.SetWithTTL([]byte(key), value, 20*time.Millisecond); err != nil {
				return false
			}
			ok, err := db.Persist([]byte(key))
			if err != nil || !ok {
				return false
			}
			remaining, err := db.TTL([]byte(key))
			if err != nil {
				return false
			}
			return remaining == -1
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
