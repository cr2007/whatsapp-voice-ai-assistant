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
	"syscall"

	"github.com/cr2007/whatsapp-whisper-groq/groq"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type transcriptionJSONBody struct {
	Transcription string `json:"transcription"`
	Language      string `json:"language"`
}

// log is a global logger instance used for logging throughout the application.
var log waLog.Logger

// quitter is a channel used to signal termination of the application.
var quitter = make(chan struct{})

// messageHead is a command-line flag that specifies the text to start each message with.
var messageHead = flag.String("message-head", "Transcript:\n> ", "Text to start message with")

func main() {
	fmt.Println("Starting main function")

	// Initialize database logger with DEBUG level logging to stdout.
	log = waLog.Stdout("Database", "DEBUG", true)

	// Create a new SQL store container using SQLite3.
	container, err := sqlstore.New("sqlite3", "file:whatsmeow.db?_foreign_keys=on", log)
	if err != nil {
		fmt.Println("Error creating SQL store container:", err)
		panic(err)
	}

	// Get the first device from the store.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	// Initialize client logger with INFO level logging to stdout.
	clientLog := waLog.Stdout("Client", "INFO", true)

	// Create a new WhatsApp client with the device store and client logger.
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Add an event handler to the client.
	client.AddEventHandler(GetEventHandler(client))

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}

		// Handle QR code events for new login.
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code in the terminal.
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal:
				// fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Disconnect the client gracefully on termination.
	client.Disconnect()
	fmt.Println("Client disconnected")
}

/*
GetEventHandler returns a function that handles various WhatsApp events.

It takes a WhatsApp client as an argument and returns a function that processes
events based on their type.

The returned function handles the following events:
  - StreamReplaced and Disconnected: Logs the event and closes the quitter channel.
  - Message: Processes audio messages, downloads the audio, and sends a transcription
    if the audio is a PTT (Push-To-Talk) message.
*/
func GetEventHandler(client *whatsmeow.Client) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.StreamReplaced, *events.Disconnected:
			// Log the event and terminate the application by closing the quitter channel.
			log.Infof("Got %+v, Terminating", evt)
			close(quitter)
		case *events.Message:
			// Get the audio message from the event.
			am := v.Message.GetAudioMessage()

			if am != nil {
				fmt.Println("Audio message received")

				// Download the audio data.
				audioData, err := client.Download(am)
				if err != nil {
					// Log an error if the download fails.
					log.Errorf("Failed to download audio: %v", err)
					return
				}

				// Print the sender's username.
				fmt.Printf("The user name is: %s\n", v.Info.Sender.User)

				if am.GetPTT() {
					// Get the transcription of the audio data.
					maybeText := getTranscription(audioData)

					if maybeText != nil {
						fmt.Println("Transcription received")

						text := *maybeText

						// Create a new message with the transcription.
						msg := &waProto.Message{
							// Create an extended text message with the transcription.
							ExtendedTextMessage: &waProto.ExtendedTextMessage{
								// Set the text of the message.
								Text: proto.String(*messageHead + text),

								// Set the context info of the message.
								ContextInfo: &waProto.ContextInfo{
									StanzaID:      proto.String(v.Info.ID),
									Participant:   proto.String(v.Info.Sender.ToNonAD().String()),
									QuotedMessage: v.Message,
								},
							},
						}

						// Send the message.
						_, err := client.SendMessage(context.Background(), v.Info.MessageSource.Chat, msg)
						if err != nil {
							fmt.Println("Error sending message:", err)
						} else {
							fmt.Println("Message sent successfully")
						}

						if os.Getenv("GROQ_API_KEY") == "" {
							fmt.Println("Groq API Key not found.\n Get your Groq API Key from https://console.groq.com to use this feature.")
						} else {
							groq.SendGroqMessage(client, text, v)
						}
					} else {
						fmt.Println("Transcription is nil")
					}
				}
			}
		}
	}
}

func getTranscription(audioData []byte) *string {
	fmt.Println("Starting transcription request")

	// Create a new POST request with the audio data as the body.
	req, err := http.NewRequest("POST", "http://127.0.0.1:5000/transcribe", bytes.NewReader(audioData))
	if err != nil {
		// Log an error and return nil if the request creation fails.
		log.Errorf("Failed to create request: %v", err)
		return nil
	}

	fmt.Println("Request created")

	// Set the Content-Type header to indicate binary data.
	req.Header.Set("Content-Type", "application/octet-stream")

	// Send the request using the default HTTP client.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Failed to send request: %v", err)
		return nil
	}
	defer resp.Body.Close()

	fmt.Println("Request sent, awaiting response")

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Log an error and return nil if reading the response body fails.
		log.Errorf("Failed to read response body: %v", err)
		return nil
	}

	fmt.Println("Response received")

	var jsonBody transcriptionJSONBody
	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		return nil
	}

	// Convert the response body to a string and return it.
	transcription := jsonBody.Transcription
	fmt.Println("Transcription:", transcription)

	return &transcription
}
