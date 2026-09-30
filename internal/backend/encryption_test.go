package backend

import (
	"bytes"
	"testing"
)

func TestSecretBoxSealAndOpen(t *testing.T) {
	box := NewSecretBox([]byte("test master key"))
	plaintext := []byte(`{"source_url":"https://timetable.canterbury.ac.nz/ical/course.ics"}`)

	payload, err := box.Seal("feed-row", plaintext)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	got, err := box.Open("feed-row", payload)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("Open() = %q, want %q", got, plaintext)
	}
}

func TestSecretBoxOpenRejectsUnauthenticatedPayloads(t *testing.T) {
	box := NewSecretBox([]byte("test master key"))
	payload, err := box.Seal("feed-row", []byte("secret"))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	modified := append([]byte(nil), payload...)
	modified[len(modified)-1] ^= 1

	tests := []struct {
		name    string
		rowID   string
		payload []byte
	}{
		{name: "different row identifier", rowID: "another-row", payload: payload},
		{name: "modified ciphertext", rowID: "feed-row", payload: modified},
		{name: "too short", rowID: "feed-row", payload: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := box.Open(test.rowID, test.payload); err == nil {
				t.Fatal("Open() error = nil, want authentication or payload error")
			}
		})
	}
}
