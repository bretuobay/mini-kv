package minikv

import (
	"time"

	"github.com/bretuobay/mini-kv/internal/wal"
)

// Batch buffers write operations for atomic commit.
type Batch interface {
	Set(key, value []byte)
	SetWithTTL(key, value []byte, ttl time.Duration)
	Delete(key []byte)
	Write() error
	Discard()
}

type batchOpType uint8

const (
	batchSet batchOpType = iota + 1
	batchDelete
)

type batchOp struct {
	opType    batchOpType
	key       []byte
	value     []byte
	expiresAt int64
}

type batchImpl struct {
	db     *DB
	opList []batchOp
	size   int64
	closed bool
	err    error
}

// NewBatch creates a new batch bound to the DB.
func (db *DB) NewBatch() Batch {
	return &batchImpl{db: db}
}

// Set buffers a Set operation.
func (b *batchImpl) Set(key, value []byte) {
	b.addOp(batchSet, key, value, -1)
}

// SetWithTTL buffers a Set operation with TTL.
func (b *batchImpl) SetWithTTL(key, value []byte, ttl time.Duration) {
	if ttl <= 0 {
		b.addOp(batchSet, key, value, -1)
		return
	}
	expiresAt := time.Now().Add(ttl).UnixNano()
	b.addOp(batchSet, key, value, expiresAt)
}

// Delete buffers a Delete operation.
func (b *batchImpl) Delete(key []byte) {
	b.addOp(batchDelete, key, nil, -1)
}

// Write applies all operations atomically.
func (b *batchImpl) Write() error {
	if b.closed {
		return ErrClosed
	}
	if b.err != nil {
		return b.err
	}
	if len(b.opList) == 0 {
		return nil
	}

	db := b.db
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}
	if db.opts.ReadOnly {
		return ErrReadOnly
	}
	if b.size > int64(db.opts.MaxBatchSize) {
		return ErrBatchTooBig
	}

	stats := db.statsOrInit()
	start := time.Now()
	now := time.Now().UnixNano()
	encoded := make([][]byte, 0, len(b.opList))
	for _, op := range b.opList {
		record := wal.WALRecord{
			Timestamp: now,
			Key:       append([]byte(nil), op.key...),
			Value:     append([]byte(nil), op.value...),
			ExpiresAt: op.expiresAt,
		}
		switch op.opType {
		case batchSet:
			record.Type = wal.RecordSet
		case batchDelete:
			record.Type = wal.RecordDelete
			record.Value = nil
			record.ExpiresAt = -1
		}
		encoded = append(encoded, wal.EncodeWALRecord(record))
	}

	for _, payload := range encoded {
		if _, err := db.wal.AppendRaw(payload); err != nil {
			stats.writes.Add(uint64(countOps(b.opList, batchSet)))
			stats.deletes.Add(uint64(countOps(b.opList, batchDelete)))
			stats.writeLatency.add(time.Since(start))
			return err
		}
	}
	if db.opts.SyncMode == SyncAlways {
		if err := db.wal.Sync(); err != nil {
			stats.writes.Add(uint64(countOps(b.opList, batchSet)))
			stats.deletes.Add(uint64(countOps(b.opList, batchDelete)))
			stats.writeLatency.add(time.Since(start))
			return err
		}
	}

	for _, op := range b.opList {
		switch op.opType {
		case batchSet:
			db.index.SetEntry(string(op.key), op.value, op.expiresAt, now)
		case batchDelete:
			db.index.Delete(string(op.key))
		}
	}
	stats.writes.Add(uint64(countOps(b.opList, batchSet)))
	stats.deletes.Add(uint64(countOps(b.opList, batchDelete)))
	stats.writeLatency.add(time.Since(start))
	b.closed = true
	return nil
}

// Discard abandons buffered operations.
func (b *batchImpl) Discard() {
	b.closed = true
	b.opList = nil
	b.size = 0
}

func (b *batchImpl) addOp(opType batchOpType, key []byte, value []byte, expiresAt int64) {
	if b.closed {
		return
	}
	if len(key) == 0 {
		return
	}
	if len(key) > b.db.opts.MaxKeySize {
		b.err = ErrKeyTooLarge
		return
	}
	if len(value) > b.db.opts.MaxValueSize {
		b.err = ErrValueTooLarge
		return
	}
	keyCopy := append([]byte(nil), key...)
	valueCopy := append([]byte(nil), value...)
	b.opList = append(b.opList, batchOp{
		opType:    opType,
		key:       keyCopy,
		value:     valueCopy,
		expiresAt: expiresAt,
	})
	b.size += int64(len(keyCopy) + len(valueCopy))
}

func countOps(ops []batchOp, opType batchOpType) int {
	count := 0
	for _, op := range ops {
		if op.opType == opType {
			count++
		}
	}
	return count
}
