package index

import (
	"bytes"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestMemIndexSetGetRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("set-get round trip", prop.ForAll(
		func(key string, value []byte) bool {
			idx := NewMemIndex()
			idx.Set(key, value, -1)
			entry, ok := idx.Get(key)
			if !ok {
				return false
			}
			return entry.ExpiresAt == -1 && bytes.Equal(entry.Value, value)
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}

func TestMemIndexGetNonExistentKey(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("get non-existent key returns false", prop.ForAll(
		func(key string) bool {
			idx := NewMemIndex()
			_, ok := idx.Get(key)
			return !ok
		},
		gen.AnyString(),
	))

	properties.TestingRun(t)
}

func TestMemIndexDeleteMakesKeyNonExistent(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("delete removes key", prop.ForAll(
		func(key string, value []byte) bool {
			idx := NewMemIndex()
			idx.Set(key, value, -1)
			idx.Delete(key)
			_, ok := idx.Get(key)
			return !ok
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}

func TestMemIndexDeleteIsIdempotent(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("delete is idempotent", prop.ForAll(
		func(key string, value []byte) bool {
			idx := NewMemIndex()
			idx.Set(key, value, -1)
			idx.Delete(key)
			idx.Delete(key)
			_, ok := idx.Get(key)
			return !ok
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}

func TestMemIndexExistsReflectsKeyState(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("exists reflects key state", prop.ForAll(
		func(key string, value []byte) bool {
			idx := NewMemIndex()
			if idx.Exists(key) {
				return false
			}
			idx.Set(key, value, -1)
			if !idx.Exists(key) {
				return false
			}
			idx.Delete(key)
			return !idx.Exists(key)
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}
