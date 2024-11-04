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

// ApiResponse represents a generic API response structure.
// It contains a single field 'Response' which holds the response message.
type ApiResponse struct {
	Response string `json:"response"`
}

// RequestBody represents the structure of a request body sent to the API.
// It contains the user ID and the message to be sent.
type RequestBody struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// RequestPayload represents the payload structure for sending messages to the Groq API.
// It contains an array of messages and the model to be used.
type RequestPayload struct {
	Messages []Message `json:"messages"`
	Model    string    `json:"model"`
}

// Message represents a single message in the request payload.
// It contains the role of the message sender and the content of the message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ApiResponseGroq represents the response structure from the Groq API.
// It contains an array of choices, each with a message containing the content.
type ApiResponseGroq struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// htmlToWhatsAppFormat converts HTML content to a format suitable for WhatsApp messages.
// It replaces certain HTML tags with WhatsApp-friendly formatting.
//
// Parameters:
//   - html: A string containing the HTML content.
//
// Returns:
//   - A string with the HTML content converted to WhatsApp-friendly formatting.
func htmlToWhatsAppFormat(html string) string {
	// Replace closing paragraph tags followed by a newline with a newline
	html = strings.ReplaceAll(html, "</p>\n", "\n")
	// Replace closing paragraph tags with a newline
	html = strings.ReplaceAll(html, "</p>", "\n")
	// Remove opening paragraph tags
	html = strings.ReplaceAll(html, "<p>", "")
	// Replace opening ordered list tags with a dash and a space
	html = strings.ReplaceAll(html, "<ol>", "- ")
	// Replace closing ordered list tags with a newline
	html = strings.ReplaceAll(html, "</ol>", "\n")
	// Replace opening list item tags with a dash and a space
	html = strings.ReplaceAll(html, "<li>", "- ")
	// Replace closing list item tags with a newline
	html = strings.ReplaceAll(html, "</li>", "\n")
	// Replace line break tags with a newline
	html = strings.ReplaceAll(html, "<br>", "\n")

	return html
}

// SendGroqMessage sends a message to Groq, formats the response, and sends it back to the user on WhatsApp.
//
// Parameters:
//   - client: A pointer to the WhatsApp client.
//   - text: The text message to send to Groq.
//   - message: A pointer to the original WhatsApp message event.
//
// This function performs the following steps:
//  1. Logs the message being sent to Groq.
//  2. Sends the user's message to Groq and retrieves the response.
//  3. Formats the response to be suitable for WhatsApp.
//  4. Constructs a WhatsApp message with the formatted response.
//  5. Sends the constructed message back to the user on WhatsApp.
func SendGroqMessage(
	client *whatsmeow.Client,
	text string,
	message *events.Message,
	contextInfo *waProto.ContextInfo,
) {
	fmt.Println("Sending message to Groq:", text)

	// Send the user's message to Groq and get the response.
	response, err := sendPostRequestGroq(text)
	if err != nil {
		panic(err)
	}

	// Format the response to WhatsApp format.
	response = htmlToWhatsAppFormat(response)

	// Construct the WhatsApp message with the formatted response.
	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("Response from Groq:\n" + response),

			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(contextInfo.GetStanzaID()),
				Participant:   proto.String(contextInfo.GetParticipant()),
				QuotedMessage: contextInfo.GetQuotedMessage(),
			},
		},
	}

	// Send the constructed message back to the user on WhatsApp.
	_, err = client.SendMessage(context.Background(), message.Info.MessageSource.Chat, msg)
	if err != nil {
		fmt.Println("Error sending message:", err)
	} else {
		fmt.Println("Message sent successfully")
	}
}

// sendPostRequestGroq sends a POST request to the Groq API with the user's message and returns the response.
//
// Parameters:
//   - text: The text message to send to the Groq API.
//
// Returns:
//   - A string containing the response from the Groq API.
//   - An error if any issue occurs during the request or response processing.
//
// This function performs the following steps:
//  1. Creates a new RequestPayload with the user's message.
//  2. Marshals the payload into a JSON body.
//  3. Creates a new POST request with the JSON body.
//  4. Sets the necessary headers, including the Groq API Key from environment variables.
//  5. Sends the request and reads the response body.
//  6. Unmarshals the response JSON into an ApiResponseGroq struct.
//  7. Returns the content of the first choice in the response or an error if no choices are found.
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

	// Gets the Groq API Key from the environment variables.
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("groq API Key not found.\n Get your Groq API Key from https://console.groq.com to use this feature")
	}

	// Set the necessary headers.
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Send the request and get the response.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// Unmarshal the response JSON into an ApiResponseGroq struct.
	var apiResponse ApiResponseGroq
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return "", fmt.Errorf("error unmarshaling response JSON: %w", err)
	}

	// Return the content of the first choice in the response.
	if len(apiResponse.Choices) > 0 {
		return apiResponse.Choices[0].Message.Content, nil
	}

	// Return an error if the response is empty.
	return "", fmt.Errorf("no choices found in response")
}
