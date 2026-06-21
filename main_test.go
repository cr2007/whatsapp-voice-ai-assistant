package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTrigger(t *testing.T) {
	tests := []struct {
		msg  string
		want trigger
	}{
		{"1> transcribe", triggerGroq},
		{"please 1> transcribe this", triggerGroq},
		{"2> transcribe", triggerOnly},
		{"2> transcribe this audio", triggerOnly},
		{"transcribe", triggerNone},     // no prefix
		{"1>", triggerNone},             // no "transcribe" keyword
		{"2>", triggerNone},             // no "transcribe" keyword
		{"3> transcribe", triggerNone},  // unrecognised prefix
		{"", triggerNone},
	}
	for _, tt := range tests {
		got := parseTrigger(tt.msg)
		if got != tt.want {
			t.Errorf("parseTrigger(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

func TestGetTranscription(t *testing.T) {
	t.Run("returns transcription on success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(transcriptionResponse{Transcription: "hello world", Language: "en"})
		}))
		defer srv.Close()

		orig := transcribeURL
		transcribeURL = srv.URL
		t.Cleanup(func() { transcribeURL = orig })

		got, err := getTranscription([]byte("audio"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Errorf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("returns empty string when transcription field is empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(transcriptionResponse{})
		}))
		defer srv.Close()

		orig := transcribeURL
		transcribeURL = srv.URL
		t.Cleanup(func() { transcribeURL = orig })

		got, err := getTranscription([]byte("audio"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("errors on invalid JSON", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		orig := transcribeURL
		transcribeURL = srv.URL
		t.Cleanup(func() { transcribeURL = orig })

		if _, err := getTranscription([]byte("audio")); err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})

	t.Run("errors when server is unreachable", func(t *testing.T) {
		orig := transcribeURL
		transcribeURL = "http://127.0.0.1:1"
		t.Cleanup(func() { transcribeURL = orig })

		if _, err := getTranscription([]byte("audio")); err == nil {
			t.Error("expected error for unreachable server, got nil")
		}
	})
}
