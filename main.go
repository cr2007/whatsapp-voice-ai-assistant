package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cr2007/whatsapp-voice-ai-assistant/groq"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mdp/qrterminal/v3"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type transcriptionResponse struct {
	Transcription string `json:"transcription"`
	Language      string `json:"language"`
}

// trigger represents the command mode parsed from an incoming message.
type trigger int

const (
	triggerNone trigger = iota
	triggerGroq         // "1> transcribe": transcribe + send to Groq
	triggerOnly         // "2> transcribe": transcribe only
)

var (
	log     waLog.Logger
	quitter = make(chan struct{})

	messageHead   = flag.String("message-head", "*Transcript:*\n> ", "text prepended to each transcription reply")
	transcribeURL = "http://127.0.0.1:5000/transcribe"
)

func main() {
	flag.Parse()

	if u := os.Getenv("TRANSCRIBE_URL"); u != "" {
		transcribeURL = u
	}

	log = waLog.Stdout("Database", "DEBUG", true)
	ctx := context.Background()

	container, err := sqlstore.New(ctx, "sqlite", "file:whatsmeow.db?_pragma=foreign_keys(ON)", log)
	if err != nil {
		fmt.Println("Error creating SQL store container:", err)
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))
	client.AddEventHandler(GetEventHandler(client, ctx))

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		if err = client.Connect(); err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		if err = client.Connect(); err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
	fmt.Println("Client disconnected")
}

// parseTrigger returns the trigger mode encoded in msg, or triggerNone.
func parseTrigger(msg string) trigger {
	if !strings.Contains(msg, "transcribe") {
		return triggerNone
	}
	if strings.Contains(msg, "1>") {
		return triggerGroq
	}
	if strings.Contains(msg, "2>") {
		return triggerOnly
	}
	return triggerNone
}

// GetEventHandler returns an event handler for the WhatsApp client.
//
// Trigger commands (sent as a reply to a quoted audio message):
//   - "1> transcribe" - transcribe audio and send result to Groq
//   - "2> transcribe" - transcribe audio only, no Groq call
func GetEventHandler(client *whatsmeow.Client, ctx context.Context) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.StreamReplaced, *events.Disconnected:
			log.Infof("Got %+v, Terminating", evt)
			close(quitter)

		case *events.Message:
			var messageBody string
			var quotedAudio *waProto.AudioMessage
			var contextInfo *waProto.ContextInfo
			hasQuotedMessage := false

			if v.Message.GetConversation() != "" {
				messageBody = v.Message.GetConversation()
			} else if v.Message.ExtendedTextMessage != nil {
				messageBody = v.Message.ExtendedTextMessage.GetText()
				if ci := v.Message.ExtendedTextMessage.GetContextInfo(); ci != nil {
					contextInfo = ci
					if quotedMsg := ci.QuotedMessage; quotedMsg != nil {
						hasQuotedMessage = true
						quotedAudio = quotedMsg.GetAudioMessage()
					}
				}
			}

			t := parseTrigger(messageBody)
			if t == triggerNone {
				return
			}

			if quotedAudio != nil {
				processAudio(client, ctx, v, quotedAudio, contextInfo, t == triggerGroq)
				return
			}

			var hint string
			if !hasQuotedMessage {
				hint = "Please reply to a voice note or audio file to use this command."
			} else {
				hint = "The quoted message is not a voice note or audio file."
			}
			if _, err := client.SendMessage(context.Background(), v.Info.MessageSource.Chat, &waProto.Message{
				Conversation: proto.String(hint),
			}); err != nil {
				fmt.Println("Error sending usage hint:", err)
			}
		}
	}
}

func processAudio(
	client *whatsmeow.Client,
	ctx context.Context,
	v *events.Message,
	audio *waProto.AudioMessage,
	contextInfo *waProto.ContextInfo,
	sendToGroq bool,
) {
	fmt.Println("Audio message received")

	resp, err := client.SendMessage(context.Background(), v.Info.MessageSource.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("Transcribing..."),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(contextInfo.GetStanzaID()),
				Participant:   proto.String(contextInfo.GetParticipant()),
				QuotedMessage: contextInfo.GetQuotedMessage(),
			},
		},
	})
	if err != nil {
		fmt.Println("Error sending 'Transcribing...' message:", err)
		return
	}

	audioData, err := client.Download(ctx, audio)
	if err != nil {
		log.Errorf("Failed to download audio: %v", err)
		return
	}

	fmt.Printf("Sender: %s\n", v.Info.Sender.User)

	text, err := getTranscription(audioData)
	if err != nil {
		log.Errorf("Transcription failed: %v", err)
		return
	}
	if text == "" {
		fmt.Println("Empty transcription received")
		return
	}

	fmt.Println("Transcription received:", text)

	editMsg := client.BuildEdit(v.Info.MessageSource.Chat, resp.ID, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(*messageHead + text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(contextInfo.GetStanzaID()),
				Participant:   proto.String(contextInfo.GetParticipant()),
				QuotedMessage: contextInfo.GetQuotedMessage(),
			},
		},
	})

	if _, err = client.SendMessage(context.Background(), v.Info.MessageSource.Chat, editMsg); err != nil {
		fmt.Println("Error editing message with transcription:", err)
	} else {
		fmt.Println("Message edited successfully with transcription")
	}

	if sendToGroq {
		if os.Getenv("GROQ_API_KEY") == "" {
			fmt.Println("Groq API Key not found. Set GROQ_API_KEY to enable AI responses.")
		} else {
			groq.SendGroqMessage(client, text, v, contextInfo)
		}
	}
}

func getTranscription(audioData []byte) (string, error) {
	req, err := http.NewRequest("POST", transcribeURL, bytes.NewReader(audioData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result transcriptionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return result.Transcription, nil
}
