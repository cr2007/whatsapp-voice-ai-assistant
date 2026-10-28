package main

import (
	"context"
	"slices"
	"testing"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// fakeClient records the messages the bot sends.
type fakeClient struct {
	sent     []*waProto.Message
	editedID types.MessageID
}

func (f *fakeClient) SendMessage(_ context.Context, _ types.JID, msg *waProto.Message, _ ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	f.sent = append(f.sent, msg)
	return whatsmeow.SendResponse{ID: "placeholder-id"}, nil
}

func (f *fakeClient) Download(context.Context, whatsmeow.DownloadableMessage) ([]byte, error) {
	return []byte("audio"), nil
}

func (f *fakeClient) BuildEdit(_ types.JID, id types.MessageID, newContent *waProto.Message) *waProto.Message {
	f.editedID = id
	return newContent
}

// texts returns the text of each sent message.
func (f *fakeClient) texts() []string {
	var out []string
	for _, m := range f.sent {
		text, _ := messageText(m)
		out = append(out, text)
	}
	return out
}

func TestParseTrigger(t *testing.T) {
	tests := []struct {
		msg  string
		want trigger
	}{
		{"1> transcribe", triggerGroq},
		{"please 2> transcribe this", triggerOnly},
		{"1> 2> transcribe", triggerGroq},
		{"transcribe", triggerNone},
		{"1>", triggerNone},
		{"3> transcribe", triggerNone},
	}
	for _, tt := range tests {
		if got := parseTrigger(tt.msg); got != tt.want {
			t.Errorf("parseTrigger(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

func TestHTMLToWhatsApp(t *testing.T) {
	in := "<p>Steps:</p>\n<ol><li>one</li><li>two</li></ol>line<br>break"
	want := "Steps:\n- - one\n- two\n\nline\nbreak"
	if got := htmlToWhatsApp(in); got != want {
		t.Errorf("htmlToWhatsApp(%q) = %q, want %q", in, got, want)
	}
}

func TestHandleMessage(t *testing.T) {
	audioQuote := &waProto.Message{AudioMessage: &waProto.AudioMessage{}}
	textQuote := &waProto.Message{Conversation: proto.String("hi")}

	tests := []struct {
		name       string
		text       string
		quoted     *waProto.Message
		transcript string
		hasGroqKey bool
		want       []string
	}{
		{"no trigger", "hello", audioQuote, "hello", true, nil},
		{"trigger without quote", "2> transcribe", nil, "", false, []string{hintNoQuote}},
		{"trigger quotes text", "2> transcribe", textQuote, "", false, []string{hintNotAudio}},
		{"transcribe only", "2> transcribe", audioQuote, "hello", true, []string{"Transcribing...", "HEAD hello"}},
		{"empty transcript", "1> transcribe", audioQuote, "", true, []string{"Transcribing..."}},
		{"groq without key", "1> transcribe", audioQuote, "hello", false, []string{"Transcribing...", "HEAD hello"}},
		{"groq", "1> transcribe", audioQuote, "hello", true, []string{"Transcribing...", "HEAD hello", "Response from Groq:\nanswer\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{}
			var asked []string
			b := &bot{
				client:      client,
				log:         waLog.Noop,
				messageHead: "HEAD ",
				transcribe: func(context.Context, []byte) (string, error) {
					return tt.transcript, nil
				},
			}
			if tt.hasGroqKey {
				b.ask = func(_ context.Context, prompt string) (string, error) {
					asked = append(asked, prompt)
					return "<p>answer</p>", nil
				}
			}

			msg := &waProto.Message{Conversation: proto.String(tt.text)}
			if tt.quoted != nil {
				msg = &waProto.Message{ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text:        proto.String(tt.text),
					ContextInfo: &waProto.ContextInfo{StanzaID: proto.String("quoted-id"), QuotedMessage: tt.quoted},
				}}
			}
			b.handleEvent(&events.Message{Message: msg})

			if got := client.texts(); !slices.Equal(got, tt.want) {
				t.Fatalf("sent %q, want %q", got, tt.want)
			}
			if len(tt.want) > 1 && client.editedID != "placeholder-id" {
				t.Errorf("edited message %q, want %q", client.editedID, "placeholder-id")
			}
			for _, m := range client.sent[min(1, len(client.sent)):] {
				if _, ci := messageText(m); ci.GetStanzaID() != "quoted-id" {
					t.Errorf("reply %q does not quote the audio message", m)
				}
			}
			if wantAsk := len(tt.want) == 3; wantAsk != (len(asked) == 1) {
				t.Errorf("Groq prompts = %q, want one call: %v", asked, wantAsk)
			}
		})
	}
}

// Regression: the handler closed a shared channel on every disconnect, so the
// second disconnect panicked with "close of closed channel".
func TestHandleEventRepeatedDisconnect(t *testing.T) {
	b := &bot{log: waLog.Noop}
	b.handleEvent(&events.Disconnected{})
	b.handleEvent(&events.Disconnected{})
	b.handleEvent(&events.StreamReplaced{})
}
