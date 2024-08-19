package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// log is a global logger instance used for logging throughout the application.
var log waLog.Logger

// quitter is a channel used to signal termination of the application.
var quitter = make(chan struct{})

// messageHead is a command-line flag that specifies the text to start each message with.
var messageHead = flag.String("message-head", "Transcript:\n> ", "Text to start message with")

func main() {
	// Initialize database logger with DEBUG level logging to stdout.
	log = waLog.Stdout("Database", "DEBUG", true)

	// Create a new SQL store container using SQLite3.
	container, err := sqlstore.New("sqlite3", "file:whatsmeow.db?_foreign_keys=on", log)
	if err != nil {
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
					maybeText := getTranscription(audioData) // TODO: Implement function

					if maybeText != nil {
						text := *maybeText

						// Create a new message with the transcription.
						msg := &waProto.Message{
							ExtendedTextMessage: &waProto.ExtendedTextMessage{
								Text: proto.String(*messageHead + text),
								ContextInfo: &waProto.ContextInfo{
									StanzaID:      proto.String(v.Info.ID),
									Participant:   proto.String(v.Info.Sender.ToNonAD().String()),
									QuotedMessage: v.Message,
								},
							},
						}

						// Send the message.
						_, _ = client.SendMessage(context.Background(), v.Info.MessageSource.Chat, msg)
					}
				}
			}
		}
	}
}

// TODO: Implement this function to get transcription from audio data
func getTranscription(audioDate []byte) *string {
	responseBody := "Transcribe to Whisper"

	return &responseBody
}
