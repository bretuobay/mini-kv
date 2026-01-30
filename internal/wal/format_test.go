package wal

import (
	"bytes"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestWALRecordEncodeDecodeRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("encode-decode round trip", prop.ForAll(
		func(recType uint8, ts int64, expiresAt int64, key []byte, value []byte) bool {
			if len(key) == 0 {
				key = []byte{0}
			}
			if recType == 0 {
				recType = uint8(RecordSet)
			}
			record := WALRecord{
				Type:      RecordType(recType),
				Timestamp: ts,
				ExpiresAt: expiresAt,
				Key:       key,
				Value:     value,
			}

			encoded := EncodeWALRecord(record)
			decoded, consumed, err := DecodeWALRecord(encoded)
			if err != nil {
				return false
			}
			if consumed != len(encoded) {
				return false
			}
			if decoded.Type != record.Type || decoded.Timestamp != record.Timestamp || decoded.ExpiresAt != record.ExpiresAt {
				return false
			}
			if !bytes.Equal(decoded.Key, record.Key) || !bytes.Equal(decoded.Value, record.Value) {
				return false
			}
			return true
		},
		gen.UInt8(),
		gen.Int64(),
		gen.Int64(),
		gen.SliceOf(gen.UInt8()),
		gen.SliceOf(gen.UInt8()),
	))

	properties.TestingRun(t)
}

func TestWALChecksumDetectsCorruption(t *testing.T) {
	record := WALRecord{
		Type:      RecordSet,
		Timestamp: 1,
		ExpiresAt: -1,
		Key:       []byte("key"),
		Value:     []byte("value"),
	}

	encoded := EncodeWALRecord(record)
	// Flip a byte in the payload to corrupt checksum.
	if len(encoded) > 0 {
		encoded[len(encoded)-1] ^= 0xFF
	}

	_, _, err := DecodeWALRecord(encoded)
	if err == nil {
		t.Fatalf("expected checksum error, got nil")
	}
	if err != ErrChecksumMismatch {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
}
