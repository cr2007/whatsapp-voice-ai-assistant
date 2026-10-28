// Package groq is a minimal client for the Groq chat completions API.
package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Model is the Groq model used for every completion.
const Model = "llama-3.1-8b-instant"

// apiURL is the Groq chat completions endpoint. Tests override it.
var apiURL = "https://api.groq.com/openai/v1/chat/completions"

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Messages []message `json:"messages"`
	Model    string    `json:"model"`
}

type response struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

// Complete sends prompt to Groq as a single user message and returns the
// content of the first choice.
func Complete(ctx context.Context, apiKey, prompt string) (string, error) {
	// Step 1: Encode the request body.
	body, err := json.Marshal(request{
		Messages: []message{{Role: "user", Content: prompt}},
		Model:    Model,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Step 2: Build the authenticated request.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Step 3: Send the request and reject non-200 replies.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// Step 4: Decode the reply and return the first choice.
	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", errors.New("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}
