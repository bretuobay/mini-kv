package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Manager handles snapshot creation and loading.
type Manager struct {
	dir string
}

// NewManager returns a snapshot manager rooted at dir.
func NewManager(dir string) *Manager {
	return &Manager{dir: dir}
}

// CreateSnapshot writes a snapshot file from the provided entries.
// Expired entries (expiresAt >=0 and <= now) are excluded.
func (m *Manager) CreateSnapshot(entries []Entry, version uint32, timestamp int64) (string, error) {
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return "", err
	}

	filtered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.ExpiresAt >= 0 && entry.ExpiresAt <= timestamp {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Ensure deterministic order when naming snapshots.
	seq := timestamp
	path := filepath.Join(m.dir, snapshotName(seq))
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := EncodeSnapshot(file, filtered, version, timestamp); err != nil {
		return "", err
	}

	return path, nil
}

// LoadSnapshot reads the snapshot file and returns entries.
func (m *Manager) LoadSnapshot(path string) (Header, []Entry, error) {
	return DecodeSnapshot(path)
}

// ListSnapshots returns snapshot files sorted by name.
func (m *Manager) ListSnapshots() ([]string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".snap") {
			continue
		}
		paths = append(paths, filepath.Join(m.dir, name))
	}

	sort.Strings(paths)
	return paths, nil
}

func snapshotName(seq int64) string {
	if seq < 0 {
		seq = -seq
	}
	return fmt.Sprintf("snapshot_%06d.snap", seq)
}
