package common

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestCustomEventRenderAcceptsNonStringData(t *testing.T) {
	recorder := httptest.NewRecorder()

	if err := (CustomEvent{Data: 42}).Render(recorder); err != nil {
		t.Fatalf("render non-string data: %v", err)
	}
	if got := recorder.Body.String(); got != "42" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content type: %q", got)
	}
}

func TestCustomEventRenderWritesEventBoundary(t *testing.T) {
	recorder := httptest.NewRecorder()

	if err := (CustomEvent{Data: "data: hello"}).Render(recorder); err != nil {
		t.Fatalf("render event: %v", err)
	}
	if got := recorder.Body.String(); got != "data: hello\n\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}

type failingEventWriter struct{}

func (failingEventWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func (failingEventWriter) writeString(string) (int, error) {
	return 0, errors.New("write failed")
}

func TestWriteDataPropagatesWriterError(t *testing.T) {
	if err := writeData(failingEventWriter{}, "data: hello"); err == nil {
		t.Fatal("expected writer error")
	}
}
