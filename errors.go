package minikv

import "errors"

var (
	ErrNotFound     = errors.New("minikv: not found")
	ErrKeyTooLarge  = errors.New("minikv: key too large")
	ErrValueTooLarge = errors.New("minikv: value too large")
	ErrBatchTooBig  = errors.New("minikv: batch too big")
	ErrReadOnly     = errors.New("minikv: read-only")
	ErrClosed       = errors.New("minikv: db closed")
	ErrInvalidValue = errors.New("minikv: invalid value")
	ErrCorruptWAL   = errors.New("minikv: corrupt wal")
	ErrLocked       = errors.New("minikv: database locked")
)

const (
	MaxKeySize   = 1024
	MaxValueSize = 10 * 1024 * 1024
	MaxBatchSize = 100 * 1024 * 1024
	MaxWALSize   = 256 * 1024 * 1024
)
