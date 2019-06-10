package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"

	"github.com/OloloevReal/go-blynk"
	slog "github.com/OloloevReal/go-simple-log"
)

func main() {
	slog.Printf("Blynk starting, version %s", blynk.Version)
	defer slog.Println("Go Blynk finished")

	//slog.SetOptions(slog.SetDebug, slog.SetCaller)

	auth := flag.String("auth", "", "set -auth=blynk_token")
	email := flag.String("email", "", "set -email=someemail@gmail.com")
	flag.Parse()
	_ = email

	app := blynk.NewBlynk(*auth)

	app.AddReaderHandler(10, func(pin uint, writer io.Writer) {
		r := rand.Intn(100)
		n, err := fmt.Fprint(writer, r)
		if err != nil {
			slog.Printf("[ERROR] %s", err)
		}
		log.Printf("[INFO] send to Pin: %d, bytes: %d, value: %d", pin, n, r)
	})

	app.AddWriterHandler(12, func(pin uint, reader io.Reader) {
		slog.Printf("[INFO] received from Pin: %d, buf: %s", pin, reader)
	})

	app.AddWriterHandler(14, func(pin uint, reader io.Reader) {
		slog.Printf("[INFO] received from Pin: %d, buf: %s", pin, reader)
	})

	if err := app.Connect(); err != nil {
		slog.Fatalln(err)
	}

	defer app.Disconnect()
	app.Processing()
}
