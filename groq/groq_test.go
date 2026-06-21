package groq

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHtmlToWhatsAppFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text unchanged", "hello world", "hello world"},
		{"closing p with newline", "hello</p>\nworld", "hello\nworld"},
		{"closing p without newline", "hello</p>world", "hello\nworld"},
		{"opening p removed", "<p>hello", "hello"},
		{"paragraph round-trip", "<p>hello</p>", "hello\n"},
		{"line break", "one<br>two", "one\ntwo"},
		{"list item", "<li>item</li>", "- item\n"},
		{"ordered list open", "<ol>items", "- items"},
		{"ordered list close", "items</ol>", "items\n"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToWhatsAppFormat(tt.input)
			if got != tt.want {
				t.Errorf("htmlToWhatsAppFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSendPostRequestGroq(t *testing.T) {
	t.Run("returns content from first choice", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(groqResponse{
				Choices: []groqChoice{
					{Message: groqMessage{Content: "test reply"}},
				},
			})
		}))
		defer srv.Close()

		t.Setenv("GROQ_API_KEY", "test-key")
		orig := groqAPIURL
		groqAPIURL = srv.URL
		t.Cleanup(func() { groqAPIURL = orig })

		got, err := sendPostRequestGroq("hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "test reply" {
			t.Errorf("got %q, want %q", got, "test reply")
		}
	})

	t.Run("errors on empty choices", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(groqResponse{})
		}))
		defer srv.Close()

		t.Setenv("GROQ_API_KEY", "test-key")
		orig := groqAPIURL
		groqAPIURL = srv.URL
		t.Cleanup(func() { groqAPIURL = orig })

		if _, err := sendPostRequestGroq("hello"); err == nil {
			t.Error("expected error for empty choices, got nil")
		}
	})

	t.Run("errors when GROQ_API_KEY is not set", func(t *testing.T) {
		t.Setenv("GROQ_API_KEY", "")

		if _, err := sendPostRequestGroq("hello"); err == nil {
			t.Error("expected error when API key missing, got nil")
		}
	})

	t.Run("errors on invalid JSON response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		t.Setenv("GROQ_API_KEY", "test-key")
		orig := groqAPIURL
		groqAPIURL = srv.URL
		t.Cleanup(func() { groqAPIURL = orig })

		if _, err := sendPostRequestGroq("hello"); err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})
}
