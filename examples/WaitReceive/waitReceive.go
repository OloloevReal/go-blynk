package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	blynk "github.com/OloloevReal/go-blynk"
	slog "github.com/OloloevReal/go-simple-log"
)

func main() {
	/*
		Ctrl + C to exit
	*/
	slog.Printf("Blynk starting, version %s", blynk.Version)
	defer slog.Println("Go Blynk finished")

	slog.SetOptions(slog.SetDebug)
	auth := flag.String("auth", "", "set -auth=blynk_token")
	email := flag.String("email", "", "set -email=someemail@gmail.com")
	_ = email
	flag.Parse()

	app := blynk.NewBlynk(*auth, true)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		slog.Printf("[WARN] interrupt signal")
		app.Stop()
	}()

	app.OnReadFunc = func(resp *blynk.BlynkRespose) {
		slog.Printf("[INFO] Received: %v", resp)
	}

	if err := app.Connect(); err != nil {
		slog.Fatalln(err)
	}

	go func() {
		time.Sleep(time.Second)

		if err := app.VirtualWrite(0, "7.654"); err != nil {
			slog.Println("[ERROR] Send command failed")
		}
		if err := app.VirtualWrite(1, "4.567"); err != nil {
			slog.Println("[ERROR] Send command failed")
		}
		if err := app.DigitalWrite(12, true); err != nil {
			slog.Println("[ERROR] Send command failed")
		}

		time.Sleep(time.Second)
		if err := app.VirtualRead(0, 1, 3, 2); err != nil {
			slog.Println("[ERROR] Read value failed")
		}
		if err := app.VirtualRead(4); err != nil {
			slog.Println("[ERROR] Read value failed")
		}
		if err := app.DigitalRead(12); err != nil {
			slog.Println("[ERROR] Read value failed")
		}
	}()

	app.Processing()
}
