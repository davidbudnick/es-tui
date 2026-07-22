package types

import "testing"

func TestLogWriter(t *testing.T) {
	w := NewLogWriter()
	if w.Len() != 0 {
		t.Fatal("expected empty")
	}
	n, err := w.Write([]byte(`{"level":"INFO","msg":"hi"}`))
	if err != nil || n == 0 {
		t.Fatalf("Write: n=%d err=%v", n, err)
	}
	// DEBUG filtered
	_, err = w.Write([]byte(`{"level":"DEBUG","msg":"skip"}`))
	if err != nil {
		t.Fatal(err)
	}
	if w.Len() != 1 {
		t.Fatalf("len=%d want 1", w.Len())
	}
	logs := w.GetLogs()
	if len(logs) != 1 || logs[0] != `{"level":"INFO","msg":"hi"}` {
		t.Fatalf("logs=%v", logs)
	}

	// Fill ring buffer
	for i := 0; i < MaxLogs+5; i++ {
		_, err := w.Write([]byte("line"))
		if err != nil {
			t.Fatal(err)
		}
	}
	if w.Len() != MaxLogs {
		t.Fatalf("len=%d want %d", w.Len(), MaxLogs)
	}
	if len(w.GetLogs()) != MaxLogs {
		t.Fatal("GetLogs length mismatch")
	}
}
