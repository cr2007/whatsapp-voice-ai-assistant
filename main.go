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
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var log waLog.Logger
var quitter = make(chan struct{})
var messageHead = flag.String("message-head", "Transcript:\n> ", "Text to start message with")

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	container, err := sqlstore.New("sqlite3", "file:whatsmeow.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(GetEventHandler(client))

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
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

	client.Disconnect()
}

func GetEventHandler(client *whatsmeow.Client) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.StreamReplaced, *events.Disconnected:
			log.Infof("Got %+v, Terminating", evt)
			close(quitter)
		case *events.Message:
			if v == nil || v.Info == (types.MessageInfo{}) {
				log.Warnf("Received nil message or nil message info: %+v", v)
				return
			}

			metaParts := []string{fmt.Sprintf("pushname: %s", v.Info.PushName), fmt.Sprintf("timestamp: %s", v.Info.Timestamp)}

			if v.Info.Type != "" {
				metaParts = append(metaParts, fmt.Sprintf("type: %s", v.Info.Type))
			}

			if v.Info.Category != "" {
				metaParts = append(metaParts, fmt.Sprintf("category: %s", v.Info.Category))
			}

			if v.IsViewOnce {
				metaParts = append(metaParts, "view once")
			}

			if v.IsViewOnce {
				metaParts = append(metaParts, "ephemeral")
			}

			if v.IsViewOnceV2 {
				metaParts = append(metaParts, "ephemeral (v2)")
			}

			if v.IsDocumentWithCaption {
				metaParts = append(metaParts, "document with caption")
			}

			// if v.IsEdit {
			// 	metaParts = append(metaParts, "edit")
			// }

			// if v.Info != (types.MessageInfo{}) {
			// 	if v.Info.ID != "" && v.Info.SourceString() != "" {
			// 		log.Infof("Received message %s from %s (%s)", v.Info.ID, v.Info.SourceString(), strings.Join(metaParts, ", "))
			// 	} else {
			// 		log.Warnf("Received message with incomplete info: %+v", v.Info)
			// 	}
			// } else {
			// 	log.Warnf("Received message with nil info: %+v", v)
			// }

			am := v.Message.GetAudioMessage()

			if am != nil {
				audioData, err := client.Download(am)
				if err != nil {
					log.Errorf("Failed to download audio: %v", err)
					return
				}

				if am.GetPTT() {
					maybeText := getTranscription(audioData) //TODO: Implement function

					if maybeText != nil {
						text := *maybeText

						//! Deprecated, need to refer to waE2E.Message
						msg := &waProto.Message{
							//! Deprecated, need to refer to waE2E.ExtendedTextMessage
							ExtendedTextMessage: &waProto.ExtendedTextMessage{
								Text: proto.String(*messageHead + text),
								//!
								ContextInfo: &waProto.ContextInfo{
									StanzaID:      proto.String(v.Info.ID),
									Participant:   proto.String(v.Info.Sender.ToNonAD().String()),
									QuotedMessage: v.Message,
								},
							},
						}

						_, _ = client.SendMessage(context.Background(), v.Info.MessageSource.Chat, msg)
					}
				}
			}
		}
	}
}

//TODO: Implement this function to get transcription from audio data
func getTranscription(audioDate []byte) *string {
	responseBody := "Transcribe to Whisper"

	return &responseBody
}
