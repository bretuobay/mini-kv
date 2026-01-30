package minikv

import "testing"

func TestGetIntoReusesBuffer(t *testing.T) {
	db, err := Open(DefaultOptions(t.TempDir()))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Set([]byte("k"), []byte("value")); err != nil {
		t.Fatalf("set: %v", err)
	}

	buf := make([]byte, 0, 10)
	out, err := db.GetInto(buf, []byte("k"))
	if err != nil {
		t.Fatalf("getinto: %v", err)
	}
	if string(out) != "value" {
		t.Fatalf("expected value, got %q", out)
	}
	if cap(out) != cap(buf) {
		t.Fatalf("expected buffer reuse")
	}
}
