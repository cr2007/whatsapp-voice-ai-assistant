// Package transcription is a client for the local Faster-Whisper
// transcription server (main.py).
package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DefaultURL is the transcription endpoint when TRANSCRIBE_URL is not set.
const DefaultURL = "http://127.0.0.1:5000/transcribe"

type response struct {
	Transcription string `json:"transcription"`
	Language      string `json:"language"`
}

// Transcribe sends raw audio bytes to the transcription server at url and
// returns the transcribed text. The text is empty when the audio has no speech.
func Transcribe(ctx context.Context, url string, audio []byte) (string, error) {
	// Step 1: Build the request with the audio as the raw body.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(audio))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// Step 2: Send the request and reject non-200 replies.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// Step 3: Decode the JSON reply.
	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return result.Transcription, nil
}
