// Command whatsapp-bot transcribes WhatsApp voice notes and audio files, and
// can send the transcript to Groq for an AI response.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/cr2007/whatsapp-voice-ai-assistant/groq"
	"github.com/cr2007/whatsapp-voice-ai-assistant/transcription"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mdp/qrterminal/v3"
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var messageHead = flag.String("message-head", "*Transcript:*\n> ", "text prepended to each transcription reply")

// main starts the bot and runs it until it receives SIGINT or SIGTERM.
func main() {
	flag.Parse()
	ctx := context.Background()
	log := waLog.Stdout("Main", "INFO", true)

	// Step 1: Open the device store and create the client.
	container, err := sqlstore.New(ctx, "sqlite", "file:whatsmeow.db?_pragma=foreign_keys(ON)", waLog.Stdout("Database", "DEBUG", true))
	if err != nil {
		log.Errorf("Failed to open SQL store: %v", err)
		os.Exit(1)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		log.Errorf("Failed to load device: %v", err)
		os.Exit(1)
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))

	// Step 2: Register the bot as the event handler.
	client.AddEventHandler(newBot(client, waLog.Stdout("Bot", "INFO", true)).handleEvent)

	// Step 3: Connect, and log in with a QR code on the first run.
	if err := connect(ctx, client, log); err != nil {
		log.Errorf("Failed to connect: %v", err)
		os.Exit(1)
	}

	// Step 4: Wait for a shutdown signal, then disconnect.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	client.Disconnect()
	log.Infof("Client disconnected")
}

// newBot creates a bot that uses client and reads its settings from the
// environment and the command-line flags.
func newBot(client messenger, log waLog.Logger) *bot {
	url := os.Getenv("TRANSCRIBE_URL")
	if url == "" {
		url = transcription.DefaultURL
	}

	b := &bot{
		client:      client,
		log:         log,
		messageHead: *messageHead,
		transcribe: func(ctx context.Context, audio []byte) (string, error) {
			return transcription.Transcribe(ctx, url, audio)
		},
	}
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		b.ask = func(ctx context.Context, prompt string) (string, error) {
			return groq.Complete(ctx, key, prompt)
		}
	}
	return b
}

// connect connects client to WhatsApp. When the device has no session, it
// prints QR codes to the terminal until the login completes.
func connect(ctx context.Context, client *whatsmeow.Client, log waLog.Logger) error {
	// Step 1: Reuse the stored session when there is one.
	if client.Store.ID != nil {
		return client.Connect()
	}

	// Step 2: Otherwise, get the QR channel before connecting.
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return err
	}
	if err := client.Connect(); err != nil {
		return err
	}

	// Step 3: Print each QR code until the login succeeds or fails.
	for evt := range qrChan {
		if evt.Event == "code" {
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		} else {
			log.Infof("Login event: %s", evt.Event)
		}
	}
	return nil
}
