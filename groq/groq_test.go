package groq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComplete(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    string
		wantErr bool
	}{
		{"first choice", http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"test reply"}}]}`, "test reply", false},
		{"no choices", http.StatusOK, `{"choices":[]}`, "", true},
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"bad key"}}`, "", true},
		{"invalid JSON", http.StatusOK, "not json", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotReq request
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				json.NewDecoder(r.Body).Decode(&gotReq)
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			orig := apiURL
			apiURL = srv.URL
			t.Cleanup(func() { apiURL = orig })

			got, err := Complete(context.Background(), "test-key", "hello")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if gotAuth != "Bearer test-key" {
				t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-key")
			}
			if gotReq.Model != Model || len(gotReq.Messages) != 1 || gotReq.Messages[0].Content != "hello" {
				t.Errorf("request = %+v, want one user message %q for model %q", gotReq, "hello", Model)
			}
		})
	}
}
