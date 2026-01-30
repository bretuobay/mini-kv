package benchmarks

import (
	"testing"

	"mini-kv"
)

func BenchmarkSet(b *testing.B) {
	dir := b.TempDir()
	db, err := minikv.Open(minikv.DefaultOptions(dir))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer db.Close()

	key := []byte("key")
	value := []byte("value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.Set(key, value)
	}
}

func BenchmarkGet(b *testing.B) {
	dir := b.TempDir()
	db, err := minikv.Open(minikv.DefaultOptions(dir))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer db.Close()

	key := []byte("key")
	value := []byte("value")
	_ = db.Set(key, value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = db.Get(key)
	}
}

func BenchmarkGetInto(b *testing.B) {
	dir := b.TempDir()
	db, err := minikv.Open(minikv.DefaultOptions(dir))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer db.Close()

	key := []byte("key")
	value := []byte("value")
	_ = db.Set(key, value)

	buf := make([]byte, 0, 64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ = db.GetInto(buf, key)
		_ = buf
	}
}

func BenchmarkGetIntoOnly(b *testing.B) {
	dir := b.TempDir()
	db, err := minikv.Open(minikv.DefaultOptions(dir))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer db.Close()

	key := []byte("key")
	value := []byte("value")
	_ = db.Set(key, value)

	buf := make([]byte, 0, 64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ = db.GetInto(buf, key)
		_ = buf
	}
}

func BenchmarkBatchWrite(b *testing.B) {
	dir := b.TempDir()
	db, err := minikv.Open(minikv.DefaultOptions(dir))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := db.NewBatch()
		for j := 0; j < 100; j++ {
			batch.Set([]byte("k"+intToString(j)), []byte("v"))
		}
		_ = batch.Write()
	}
}

func BenchmarkStartup(b *testing.B) {
	dir := b.TempDir()
	db, err := minikv.Open(minikv.DefaultOptions(dir))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	for i := 0; i < 1000; i++ {
		_ = db.Set([]byte("k"+intToString(i)), []byte("v"))
	}
	_ = db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db2, err := minikv.Open(minikv.DefaultOptions(dir))
		if err != nil {
			b.Fatalf("open: %v", err)
		}
		_ = db2.Close()
	}
}

func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		digit := v % 10
		buf = append(buf, byte('0'+digit))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
