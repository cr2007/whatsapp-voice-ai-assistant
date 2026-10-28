package main

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Usage hints sent when a trigger command does not quote an audio message.
const (
	hintNoQuote  = "Please reply to a voice note or audio file to use this command."
	hintNotAudio = "The quoted message is not a voice note or audio file."
)

// trigger is the command mode parsed from an incoming message.
type trigger int

const (
	triggerNone trigger = iota
	triggerGroq         // "1> transcribe": transcribe, then send to Groq
	triggerOnly         // "2> transcribe": transcribe only
)

// messenger is the subset of *whatsmeow.Client that the bot uses.
// Tests replace it with a fake.
type messenger interface {
	SendMessage(ctx context.Context, to types.JID, msg *waProto.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error)
	Download(ctx context.Context, msg whatsmeow.DownloadableMessage) ([]byte, error)
	BuildEdit(chat types.JID, id types.MessageID, newContent *waProto.Message) *waProto.Message
}

// bot handles WhatsApp events and runs the transcription flow.
type bot struct {
	client      messenger
	log         waLog.Logger
	messageHead string

	// transcribe converts audio bytes to text.
	transcribe func(ctx context.Context, audio []byte) (string, error)
	// ask sends text to the AI model. It is nil when GROQ_API_KEY is not set.
	ask func(ctx context.Context, prompt string) (string, error)
}

// parseTrigger returns the trigger mode encoded in msg, or triggerNone.
// The command can appear anywhere in the message.
func parseTrigger(msg string) trigger {
	switch {
	case !strings.Contains(msg, "transcribe"):
		return triggerNone
	case strings.Contains(msg, "1>"):
		return triggerGroq
	case strings.Contains(msg, "2>"):
		return triggerOnly
	default:
		return triggerNone
	}
}

// messageText returns the text of msg and, for an extended text message, its
// context info. The context info holds the quoted message of a reply.
func messageText(msg *waProto.Message) (string, *waProto.ContextInfo) {
	if text := msg.GetConversation(); text != "" {
		return text, nil
	}
	ext := msg.GetExtendedTextMessage()
	return ext.GetText(), ext.GetContextInfo()
}

// quotedReply builds a text message that quotes the message in ci.
func quotedReply(text string, ci *waProto.ContextInfo) *waProto.Message {
	return &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(ci.GetStanzaID()),
				Participant:   proto.String(ci.GetParticipant()),
				QuotedMessage: ci.GetQuotedMessage(),
			},
		},
	}
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

// htmlToWhatsApp converts the HTML tags that Groq can return to WhatsApp
// plain-text equivalents.
func htmlToWhatsApp(html string) string {
	return htmlReplacer.Replace(html)
}

// handleEvent is the whatsmeow event handler.
func (b *bot) handleEvent(evt any) {
	switch v := evt.(type) {
	case *events.StreamReplaced, *events.Disconnected:
		b.log.Warnf("Connection lost: %T", evt)
	case *events.Message:
		b.handleMessage(context.Background(), v)
	}
}

// handleMessage runs a trigger command when msg contains one.
//
// A trigger must quote a voice note or audio file. Otherwise the bot sends a
// usage hint to the chat.
func (b *bot) handleMessage(ctx context.Context, msg *events.Message) {
	// Step 1: Read the text and look for a trigger command.
	text, ci := messageText(msg.Message)
	t := parseTrigger(text)
	if t == triggerNone {
		return
	}

	// Step 2: Run the audio flow when the reply quotes audio.
	quoted := ci.GetQuotedMessage()
	if audio := quoted.GetAudioMessage(); audio != nil {
		b.processAudio(ctx, msg, audio, ci, t == triggerGroq)
		return
	}

	// Step 3: Otherwise, send a usage hint.
	hint := hintNotAudio
	if quoted == nil {
		hint = hintNoQuote
	}
	if _, err := b.client.SendMessage(ctx, msg.Info.Chat, &waProto.Message{Conversation: proto.String(hint)}); err != nil {
		b.log.Errorf("Failed to send usage hint: %v", err)
	}
}

// processAudio transcribes the quoted audio and replies with the transcript.
// When sendToGroq is true, it also replies with the AI response to the
// transcript.
func (b *bot) processAudio(
	ctx context.Context,
	msg *events.Message,
	audio *waProto.AudioMessage,
	ci *waProto.ContextInfo,
	sendToGroq bool,
) {
	chat := msg.Info.Chat
	b.log.Infof("Audio transcription requested by %s", msg.Info.Sender.User)

	// Step 1: Send a placeholder reply. Step 4 edits it.
	placeholder, err := b.client.SendMessage(ctx, chat, quotedReply("Transcribing...", ci))
	if err != nil {
		b.log.Errorf("Failed to send placeholder reply: %v", err)
		return
	}

	// Step 2: Download the audio.
	data, err := b.client.Download(ctx, audio)
	if err != nil {
		b.log.Errorf("Failed to download audio: %v", err)
		return
	}

	// Step 3: Transcribe the audio.
	// Step 3.1: Stop when the transcription fails or is empty.
	text, err := b.transcribe(ctx, data)
	if err != nil {
		b.log.Errorf("Transcription failed: %v", err)
		return
	}
	if text == "" {
		b.log.Warnf("Empty transcription received")
		return
	}
	b.log.Infof("Transcription received: %s", text)

	// Step 4: Replace the placeholder with the transcript.
	edit := b.client.BuildEdit(chat, placeholder.ID, quotedReply(b.messageHead+text, ci))
	if _, err := b.client.SendMessage(ctx, chat, edit); err != nil {
		b.log.Errorf("Failed to edit reply with transcription: %v", err)
	}

	// Step 5: Optionally send the transcript to Groq.
	// Step 5.1: Skip when GROQ_API_KEY is not set.
	// Step 5.2: Reply with the formatted AI response.
	if !sendToGroq {
		return
	}
	if b.ask == nil {
		b.log.Warnf("GROQ_API_KEY is not set. Set it to enable AI responses.")
		return
	}
	answer, err := b.ask(ctx, text)
	if err != nil {
		b.log.Errorf("Groq request failed: %v", err)
		return
	}
	if _, err := b.client.SendMessage(ctx, chat, quotedReply("Response from Groq:\n"+htmlToWhatsApp(answer), ci)); err != nil {
		b.log.Errorf("Failed to send Groq response: %v", err)
	}
}
