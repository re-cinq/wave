package adaptertest

import (
	"io"
	"testing"
	"time"
)

func TestSlowReader_Read(t *testing.T) {
	data := "hello world"
	reader := NewSlowReader(data, 5, 10*time.Millisecond)

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if n != 5 {
		t.Errorf("expected 5 bytes, got: %d", n)
	}

	if string(buf[:5]) != "hello" {
		t.Errorf("expected 'hello', got: %s", string(buf[:5]))
	}

	n, _ = reader.Read(buf)
	if n != 5 {
		t.Errorf("expected 5 bytes, got: %d", n)
	}
	if string(buf[:n]) != " worl" {
		t.Errorf("expected ' worl', got: %s", string(buf[:n]))
	}

	n, err = reader.Read(buf)
	if n != 1 {
		t.Errorf("expected 1 byte, got: %d", n)
	}
	if err != io.EOF {
		t.Errorf("expected EOF, got: %v", err)
	}
}
