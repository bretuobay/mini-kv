package wal

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// RecordType identifies WAL operation kinds.
type RecordType uint8

const (
	RecordSet RecordType = iota + 1
	RecordDelete
)

// WALRecord represents a single write-ahead log entry.
type WALRecord struct {
	Type      RecordType
	Timestamp int64
	Key       []byte
	Value     []byte
	ExpiresAt int64
}

var (
	ErrInvalidRecord   = errors.New("wal: invalid record")
	ErrChecksumMismatch = errors.New("wal: checksum mismatch")
)

// EncodeWALRecord encodes a record with varint lengths and CRC32 checksum.
func EncodeWALRecord(record WALRecord) []byte {
	keyLen := uint64(len(record.Key))
	valueLen := uint64(len(record.Value))

	payloadLen := 1 + 8 + 8 +
		uvarintSize(keyLen) + uvarintSize(valueLen) +
		int(keyLen) + int(valueLen) + 4
	lengthBuf := make([]byte, binary.MaxVarintLen64)
	lengthN := binary.PutUvarint(lengthBuf, uint64(payloadLen))

	payload := make([]byte, payloadLen)
	payload[0] = byte(record.Type)
	binary.LittleEndian.PutUint64(payload[1:9], uint64(record.Timestamp))
	binary.LittleEndian.PutUint64(payload[9:17], uint64(record.ExpiresAt))
	off := 17
	off += binary.PutUvarint(payload[off:], keyLen)
	off += binary.PutUvarint(payload[off:], valueLen)
	copy(payload[off:], record.Key)
	off += int(keyLen)
	copy(payload[off:], record.Value)
	off += int(valueLen)

	checksum := crc32.ChecksumIEEE(payload[:off])
	binary.LittleEndian.PutUint32(payload[off:], checksum)

	out := make([]byte, lengthN+len(payload))
	copy(out, lengthBuf[:lengthN])
	copy(out[lengthN:], payload)
	return out
}

// DecodeWALRecord decodes a record from data, returning the record and bytes consumed.
func DecodeWALRecord(data []byte) (WALRecord, int, error) {
	var rec WALRecord
	length, n := binary.Uvarint(data)
	if n <= 0 {
		return rec, 0, ErrInvalidRecord
	}
	if len(data) < n+int(length) {
		return rec, 0, ErrInvalidRecord
	}
	payload := data[n : n+int(length)]
	if len(payload) < 1+8+8+4 {
		return rec, 0, ErrInvalidRecord
	}

	rec.Type = RecordType(payload[0])
	rec.Timestamp = int64(binary.LittleEndian.Uint64(payload[1:9]))
	rec.ExpiresAt = int64(binary.LittleEndian.Uint64(payload[9:17]))
	off := 17
	keyLen, read := binary.Uvarint(payload[off:])
	if read <= 0 {
		return rec, 0, ErrInvalidRecord
	}
	off += read
	valueLen, read := binary.Uvarint(payload[off:])
	if read <= 0 {
		return rec, 0, ErrInvalidRecord
	}
	off += read

	if len(payload) < off+int(keyLen)+int(valueLen)+4 {
		return rec, 0, ErrInvalidRecord
	}
	rec.Key = append([]byte(nil), payload[off:off+int(keyLen)]...)
	off += int(keyLen)
	rec.Value = append([]byte(nil), payload[off:off+int(valueLen)]...)
	off += int(valueLen)

	storedChecksum := binary.LittleEndian.Uint32(payload[off : off+4])
	computed := crc32.ChecksumIEEE(payload[:off])
	if storedChecksum != computed {
		return rec, 0, ErrChecksumMismatch
	}

	return rec, n + int(length), nil
}

func uvarintSize(v uint64) int {
	switch {
	case v < 1<<7:
		return 1
	case v < 1<<14:
		return 2
	case v < 1<<21:
		return 3
	case v < 1<<28:
		return 4
	case v < 1<<35:
		return 5
	case v < 1<<42:
		return 6
	case v < 1<<49:
		return 7
	case v < 1<<56:
		return 8
	case v < 1<<63:
		return 9
	default:
		return 10
	}
}
