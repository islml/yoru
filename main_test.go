package main

import (
	"bytes"
	"testing"
)

func TestWriteResponse(t *testing.T) {
	var buf bytes.Buffer
	if err := writeResponse(&buf, 404, "text/plain", "Not Found"); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	want := "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nContent-Length: 9\r\nConnection: close\r\n\r\nNot Found"

	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
