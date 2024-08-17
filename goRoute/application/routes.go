package application

import (
	"net/http"

	"github.com/cr2007/whatsapp-whisper-groq/goRoute/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func loadRoutes() *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.Logger)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Route("/transcribe", loadGolangRoutes)

	return router
}

func loadGolangRoutes(router chi.Router)  {
	whatsAppHandler := &handler.WhatsApp{}

	router.Post("/", whatsAppHandler.TranscribeVoiceNote)
	router.Get("/", whatsAppHandler.TranscribeVoiceNote)
}
