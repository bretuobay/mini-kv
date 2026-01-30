package minikv

// SyncMode controls when WAL data is flushed to disk.
type SyncMode uint8

const (
	SyncAlways SyncMode = iota
	SyncPeriodic
	SyncManual
)

// Options configures database behavior.
type Options struct {
	Path         string
	ReadOnly     bool
	SyncMode     SyncMode
	MaxKeySize   int
	MaxValueSize int
	MaxBatchSize int
	MaxWALSize   int64
}

// DefaultOptions returns a baseline configuration for a database at path.
func DefaultOptions(path string) Options {
	return Options{
		Path:         path,
		SyncMode:     SyncPeriodic,
		MaxKeySize:   MaxKeySize,
		MaxValueSize: MaxValueSize,
		MaxBatchSize: MaxBatchSize,
		MaxWALSize:   MaxWALSize,
	}
}
