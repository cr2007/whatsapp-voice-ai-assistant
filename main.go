package main

import (
	"context"
	"fmt"

	"github.com/cr2007/whatsapp-whisper-groq/goRoute/application"
	// "github.com/ncruces/go-sqlite3"
)

func main() {
	app := application.New()

	err := app.Start(context.TODO())

	if err != nil {
		fmt.Println("failed to start app: ", err)
	}
}
