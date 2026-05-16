package wal

import "testing"

func TestWriterPlaceholder_BitsUT(t *testing.T) {
	writer := NewWriter(t.TempDir(), 1024)
	if writer == nil {
		t.Fatalf("writer should not be nil")
	}
	offset, err := writer.Append([]byte("hello"))
	if err != nil || offset != 0 {
		t.Fatalf("Append offset=%d err=%v", offset, err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}
	records, err := writer.Recover()
	if err != nil || records != nil {
		t.Fatalf("Recover records=%v err=%v", records, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
}
