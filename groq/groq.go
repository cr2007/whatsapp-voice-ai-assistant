package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type ApiResponse struct {
	Response string `json:"response"`
}

type RequestBody struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type RequestPayload struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ApiResponseGroq struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func htmlToWhatsAppFormat(html string) string {
	// Replace HTML tags with WhatsApp-friendly formatting
	html = strings.ReplaceAll(html, "</p>\n", "\n")
	html = strings.ReplaceAll(html, "</p>", "\n")
	html = strings.ReplaceAll(html, "<p>", "")
	html = strings.ReplaceAll(html, "<ol>", "- ")
	html = strings.ReplaceAll(html, "</ol>", "\n")
	html = strings.ReplaceAll(html, "<li>", "- ")
	html = strings.ReplaceAll(html, "</li>", "\n")
	html = strings.ReplaceAll(html, "<br>", "\n")

	return html
}

func SendGroqMessage(client *whatsmeow.Client, text string, message *events.Message) {
	fmt.Println("Sending message to Groq:", text)

	// Send the user's message to Groq and get the response.
	response, err := sendPostRequestGroq(text)
	if err != nil {
		panic(err)
	}

	// Format the response to WhatsApp format
	response = htmlToWhatsAppFormat(response)

	// Send the response to the user.
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("Response from Groq:\n" + response),

			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(message.Info.ID),
				Participant:   proto.String(message.Info.Sender.ToNonAD().String()),
				QuotedMessage: message.Message,
			},
		},
	}

	_, err = client.SendMessage(context.Background(), message.Info.MessageSource.Chat, msg)
	if err != nil {
		fmt.Println("Error sending message:", err)
	} else {
		fmt.Println("Message sent successfully")
	}

}

func sendPostRequestGroq(text string) (string, error) {
	// Create a new RequestPayload with the user's message.
	payload := RequestPayload{
		Messages: []Message{
			{
				Role:    "user",
				Content: text,
			},
		},
		Model: "llama-3.1-8b-instant",
	}

	// Marshal the payload into a JSON body.
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Create a new POST request with the JSON body.
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Gets the Groq API Key from the environment variables
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("groq API Key not found.\n Get your Groq API Key from https://console.groq.com to use this feature")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	var apiResponse ApiResponseGroq
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message.Content, nil
	}

	// Return an error if the response is empty.
	return "", fmt.Errorf("no choices found in response")
}
