package handler

import (
	"fmt"
	"net/http"
)

type WhatsApp struct{}

func (f *WhatsApp) TranscribeVoiceNote(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Transcribing Voice Note")
}
