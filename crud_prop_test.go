package minikv

import (
	"bytes"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestCRUDProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("set-get round trip", prop.ForAll(
		func(key string, value []byte) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)
			db, err := Open(opts)
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
			got, err := db.Get([]byte(key))
			if err != nil {
				return false
			}
			return bytes.Equal(got, value)
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("get non-existent key", prop.ForAll(
		func(key string) bool {
			dir := t.TempDir()
			db, err := Open(DefaultOptions(dir))
			if err != nil {
				return false
			}
			defer db.Close()
			if key == "" {
				key = "k"
			}
			_, err = db.Get([]byte(key))
			return err == ErrNotFound
		},
		gen.AnyString(),
	))

	properties.Property("delete makes key non-existent", prop.ForAll(
		func(key string, value []byte) bool {
			dir := t.TempDir()
			db, err := Open(DefaultOptions(dir))
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
			if err := db.Delete([]byte(key)); err != nil {
				return false
			}
			_, err = db.Get([]byte(key))
			return err == ErrNotFound
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("delete idempotence", prop.ForAll(
		func(key string, value []byte) bool {
			dir := t.TempDir()
			db, err := Open(DefaultOptions(dir))
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
			if err := db.Delete([]byte(key)); err != nil {
				return false
			}
			if err := db.Delete([]byte(key)); err != nil {
				return false
			}
			_, err = db.Get([]byte(key))
			return err == ErrNotFound
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("exists reflects key state", prop.ForAll(
		func(key string, value []byte) bool {
			dir := t.TempDir()
			db, err := Open(DefaultOptions(dir))
			if err != nil {
				return false
			}
			defer db.Close()
			if key == "" {
				key = "k"
			}

			exists, err := db.Exists([]byte(key))
			if err != nil || exists {
				return false
			}
			if err := db.Set([]byte(key), value); err != nil {
				return false
			}
			exists, err = db.Exists([]byte(key))
			if err != nil || !exists {
				return false
			}
			if err := db.Delete([]byte(key)); err != nil {
				return false
			}
			exists, err = db.Exists([]byte(key))
			return err == nil && !exists
		},
		gen.AnyString(),
		gen.SliceOf(gen.UInt8()),
	))

	properties.Property("key size validation", prop.ForAll(
		func(size uint16) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()
			key := make([]byte, int(size)+opts.MaxKeySize+1)
			return db.Set(key, []byte("v")) == ErrKeyTooLarge
		},
		gen.UInt16(),
	))

	properties.Property("value size validation", prop.ForAll(
		func(size uint16) bool {
			dir := t.TempDir()
			opts := DefaultOptions(dir)
			db, err := Open(opts)
			if err != nil {
				return false
			}
			defer db.Close()
			value := make([]byte, int(size)+opts.MaxValueSize+1)
			return db.Set([]byte("k"), value) == ErrValueTooLarge
		},
		gen.UInt16(),
	))

	properties.TestingRun(t)
}
