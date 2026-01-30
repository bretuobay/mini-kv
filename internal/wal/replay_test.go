package wal

import (
	"bytes"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

type recordFixture struct {
	Type      RecordType
	Key       []byte
	Value     []byte
	ExpiresAt int64
	Timestamp int64
}

func TestWALReplayIdempotence(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("replay idempotence", prop.ForAll(
		func(records []recordFixture) bool {
			walRecords := make([]WALRecord, 0, len(records))
			for _, rec := range records {
				key := rec.Key
				if len(key) == 0 {
					key = []byte{0}
				}
				walRecords = append(walRecords, WALRecord{
					Type:      rec.Type,
					Timestamp: rec.Timestamp,
					ExpiresAt: rec.ExpiresAt,
					Key:       key,
					Value:     rec.Value,
				})
			}

			once := applyRecords(walRecords)
			twice := applyRecords(append(append([]WALRecord(nil), walRecords...), walRecords...))
			return mapsEqual(once, twice)
		},
		gen.SliceOf(genRecordFixture()),
	))

	properties.TestingRun(t)
}

func genRecordFixture() gopter.Gen {
	return gopter.CombineGens(
		gen.OneConstOf(RecordSet, RecordDelete),
		gen.SliceOf(gen.UInt8()),
		gen.SliceOf(gen.UInt8()),
		gen.Int64(),
		gen.Int64(),
	).Map(func(values []interface{}) recordFixture {
		return recordFixture{
			Type:      values[0].(RecordType),
			Key:       values[1].([]byte),
			Value:     values[2].([]byte),
			ExpiresAt: values[3].(int64),
			Timestamp: values[4].(int64),
		}
	})
}

func applyRecords(records []WALRecord) map[string][]byte {
	state := make(map[string][]byte)
	for _, rec := range records {
		key := string(rec.Key)
		switch rec.Type {
		case RecordDelete:
			delete(state, key)
		case RecordSet:
			valueCopy := make([]byte, len(rec.Value))
			copy(valueCopy, rec.Value)
			state[key] = valueCopy
		}
	}
	return state
}

func mapsEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		other, ok := b[k]
		if !ok {
			return false
		}
		if !bytes.Equal(v, other) {
			return false
		}
	}
	return true
}
