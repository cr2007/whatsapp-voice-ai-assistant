package transcription

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranscribe(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{"success", http.StatusOK, `{"transcription":"hello world","language":"en"}`, "hello world", false},
		{"server error", http.StatusInternalServerError, `{"detail":"boom"}`, "", true},
		{"invalid JSON", http.StatusOK, "not json", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			got, err := Transcribe(context.Background(), srv.URL, []byte("audio"))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if string(gotBody) != "audio" {
				t.Errorf("server got body %q, want %q", gotBody, "audio")
			}
		})
	}
}
