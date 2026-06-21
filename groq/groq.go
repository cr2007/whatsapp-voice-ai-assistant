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

// groqAPIURL is the Groq chat completions endpoint. Overridable in tests.
var groqAPIURL = "https://api.groq.com/openai/v1/chat/completions"

type groqRequest struct {
	Messages []groqMessage `json:"messages"`
	Model    string        `json:"model"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqChoice struct {
	Message groqMessage `json:"message"`
}

type groqResponse struct {
	Choices []groqChoice `json:"choices"`
}

var htmlReplacer = strings.NewReplacer(
	"</p>\n", "\n",
	"</p>", "\n",
	"<p>", "",
	"<ol>", "- ",
	"</ol>", "\n",
	"<li>", "- ",
	"</li>", "\n",
	"<br>", "\n",
)

// htmlToWhatsAppFormat converts a subset of HTML tags to WhatsApp text equivalents.
func htmlToWhatsAppFormat(html string) string {
	return htmlReplacer.Replace(html)
}

// SendGroqMessage sends text to Groq and replies with the AI response in the same WhatsApp chat.
func SendGroqMessage(
	client *whatsmeow.Client,
	text string,
	message *events.Message,
	contextInfo *waProto.ContextInfo,
) {
	fmt.Println("Sending message to Groq:", text)

	response, err := sendPostRequestGroq(text)
	if err != nil {
		fmt.Println("Error sending message to Groq:", err)
		return
	}

	msg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("Response from Groq:\n" + htmlToWhatsAppFormat(response)),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(contextInfo.GetStanzaID()),
				Participant:   proto.String(contextInfo.GetParticipant()),
				QuotedMessage: contextInfo.GetQuotedMessage(),
			},
		},
	}

	_, err = client.SendMessage(context.Background(), message.Info.MessageSource.Chat, msg)
	if err != nil {
		fmt.Println("Error sending Groq response:", err)
	} else {
		fmt.Println("Groq response sent successfully")
	}
}

func sendPostRequestGroq(text string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not set")
	}

	body, err := json.Marshal(groqRequest{
		Messages: []groqMessage{{Role: "user", Content: text}},
		Model:    "llama-3.1-8b-instant",
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", groqAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result groqResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}
