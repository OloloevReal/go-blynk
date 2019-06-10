package main

import (
	"flag"

	"github.com/OloloevReal/go-blynk"
	slog "github.com/OloloevReal/go-simple-log"
)

func main() {
	slog.Printf("Blynk starting, version %s", blynk.Version)
	defer slog.Println("Go Blynk finished")

	slog.SetOptions(slog.SetDebug)
	auth := flag.String("auth", "", "set -auth=blynk_token")
	email := flag.String("email", "", "set -email=someemail@gmail.com")
	flag.Parse()
	slog.Println(*email)
	app := blynk.NewBlynk(*auth)

	if err := app.Connect(); err != nil {
		slog.Fatalln(err)
	}
	defer app.Disconnect()

	if err := app.Notify("Test push notification Test push notification Test push notification Test push notification Test push notification Test push notification"); err != nil {
		slog.Printf("[ERROR] Notify failed, %s", err)
	}

	if err := app.Tweet("Tweet message"); err != nil {
		slog.Printf("[ERROR] Notify failed, %s", err)
	}

	if err := app.EMail(*email, "test blynk", "water leak sensor enabled"); err != nil {
		slog.Printf("[ERROR] Notify failed, %s", err)
	}
}
