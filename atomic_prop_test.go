package minikv

import (
	"strconv"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestAtomicProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	properties.Property("setnx only sets once", prop.ForAll(
		func(key string, value []byte) bool {
			if key == "" {
				key = "k"
			}
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			ok, err := db.SetNX([]byte(key), value)
			if err != nil || !ok {
				return false
			}
			ok, err = db.SetNX([]byte(key), []byte("x"))
			return err == nil && !ok
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("integer operations", prop.ForAll(
		func(start int64, delta int64) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			key := []byte("c")
			_ = db.Set(key, []byte(strconv.FormatInt(start, 10)))
			val, err := db.IncrBy(key, delta)
			if err != nil {
				return false
			}
			return val == start+delta
		},
		gen.Int64(),
		gen.Int64(),
	))

	properties.Property("compare and swap", prop.ForAll(
		func(oldVal []byte, newVal []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			_ = db.Set([]byte("k"), oldVal)
			ok, err := db.CompareAndSwap([]byte("k"), oldVal, newVal)
			if err != nil || !ok {
				return false
			}
			value, _ := db.Get([]byte("k"))
			return string(value) == string(newVal)
		},
		gen.SliceOf(gen.UInt8()),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("get and set returns old value", prop.ForAll(
		func(oldVal []byte, newVal []byte) bool {
			db, err := Open(DefaultOptions(t.TempDir()))
			if err != nil {
				return false
			}
			defer db.Close()

			_ = db.Set([]byte("k"), oldVal)
			old, err := db.GetAndSet([]byte("k"), newVal)
			if err != nil {
				return false
			}
			return string(old) == string(oldVal)
		},
		gen.SliceOf(gen.UInt8()),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
