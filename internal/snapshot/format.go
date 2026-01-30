package snapshot

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"sort"
)

var (
	snapshotMagic = [8]byte{'M', 'I', 'N', 'I', 'K', 'V', 'S', 'N'}
)

// Entry is a snapshot record.
type Entry struct {
	Key       []byte
	Value     []byte
	ExpiresAt int64
	CreatedAt int64
}

// Header captures snapshot metadata.
type Header struct {
	Magic     [8]byte
	Version   uint32
	Timestamp int64
	Count     uint64
}

var (
	ErrInvalidSnapshot  = errors.New("snapshot: invalid file")
	ErrSnapshotChecksum = errors.New("snapshot: checksum mismatch")
)

// EncodeSnapshot writes entries to writer in sorted key order.
// It returns the CRC32 checksum of the payload.
func EncodeSnapshot(w io.Writer, entries []Entry, version uint32, timestamp int64) (uint32, error) {
	sorted := make([]Entry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return string(sorted[i].Key) < string(sorted[j].Key) })

	head := Header{
		Magic:     snapshotMagic,
		Version:   version,
		Timestamp: timestamp,
		Count:     uint64(len(sorted)),
	}

	buf := bufio.NewWriter(w)
	if err := writeHeader(buf, head); err != nil {
		return 0, err
	}

	hash := crc32.NewIEEE()
	multi := io.MultiWriter(buf, hash)

	for _, entry := range sorted {
		if err := writeEntry(multi, entry); err != nil {
			return 0, err
		}
	}

	checksum := hash.Sum32()
	if err := binary.Write(buf, binary.LittleEndian, checksum); err != nil {
		return 0, err
	}

	if err := buf.Flush(); err != nil {
		return 0, err
	}

	return checksum, nil
}

// DecodeSnapshot reads a snapshot file and returns header and entries.
func DecodeSnapshot(path string) (Header, []Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return Header{}, nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	head, err := readHeader(reader)
	if err != nil {
		return Header{}, nil, err
	}
	if head.Magic != snapshotMagic {
		return Header{}, nil, ErrInvalidSnapshot
	}

	entries := make([]Entry, 0, head.Count)
	hash := crc32.NewIEEE()
	multi := io.TeeReader(reader, hash)

	for i := uint64(0); i < head.Count; i++ {
		entry, err := readEntry(multi)
		if err != nil {
			return Header{}, nil, err
		}
		entries = append(entries, entry)
	}

	// After reading entries, read checksum from the underlying reader.
	var stored uint32
	if err := binary.Read(reader, binary.LittleEndian, &stored); err != nil {
		return Header{}, nil, err
	}
	computed := hash.Sum32()
	if stored != computed {
		return Header{}, nil, ErrSnapshotChecksum
	}

	return head, entries, nil
}

func writeHeader(w io.Writer, head Header) error {
	if err := binary.Write(w, binary.LittleEndian, head.Magic); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, head.Version); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, head.Timestamp); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, head.Count); err != nil {
		return err
	}
	return nil
}

func readHeader(r io.Reader) (Header, error) {
	var head Header
	if err := binary.Read(r, binary.LittleEndian, &head.Magic); err != nil {
		return head, err
	}
	if err := binary.Read(r, binary.LittleEndian, &head.Version); err != nil {
		return head, err
	}
	if err := binary.Read(r, binary.LittleEndian, &head.Timestamp); err != nil {
		return head, err
	}
	if err := binary.Read(r, binary.LittleEndian, &head.Count); err != nil {
		return head, err
	}
	return head, nil
}

func writeEntry(w io.Writer, entry Entry) error {
	if err := writeBytes(w, entry.Key); err != nil {
		return err
	}
	if err := writeBytes(w, entry.Value); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, entry.ExpiresAt); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, entry.CreatedAt); err != nil {
		return err
	}
	return nil
}

func readEntry(r io.Reader) (Entry, error) {
	key, err := readBytes(r)
	if err != nil {
		return Entry{}, err
	}
	value, err := readBytes(r)
	if err != nil {
		return Entry{}, err
	}
	var expiresAt int64
	if err := binary.Read(r, binary.LittleEndian, &expiresAt); err != nil {
		return Entry{}, err
	}
	var createdAt int64
	if err := binary.Read(r, binary.LittleEndian, &createdAt); err != nil {
		return Entry{}, err
	}
	return Entry{Key: key, Value: value, ExpiresAt: expiresAt, CreatedAt: createdAt}, nil
}

func writeBytes(w io.Writer, data []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint64(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readBytes(r io.Reader) ([]byte, error) {
	var length uint64
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}
