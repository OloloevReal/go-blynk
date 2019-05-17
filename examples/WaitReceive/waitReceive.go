package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	blynk "github.com/OloloevReal/go-blynk"
)

func main() {
	/*
		Ctrl + C to exit
	*/
	log.Printf("Blynk starting, version %s", blynk.Version)
	defer log.Printf("Go Blynk finished")

	auth := flag.String("auth", "", "set -auth=blynk_token")
	flag.Parse()

	app := blynk.NewBlynk(*auth, "blynk-cloud.com", 80, false)

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Print("[WARN] interrupt signal")
		app.Stop()
	}()

	app.OnReadFunc = func(resp *blynk.BlynkRespose) {
		log.Println("Received:", resp)
	}

	if err := app.Connect(); err != nil {
		log.Fatalln(err)
	}

	go func() {
		time.Sleep(time.Second)
		if err := app.VirtualWrite(0, "7.654"); err != nil {
			log.Println("Send command failed")
		}
		if err := app.VirtualWrite(1, "4.567"); err != nil {
			log.Println("Send command failed")
		}

		time.Sleep(time.Second)
		if err := app.VirtualRead(0, 1, 3, 2); err != nil {
			log.Println("Read value failed")
		}

		if err := app.VirtualRead(4); err != nil {
			log.Println("Read value failed")
		}
	}()

	app.Processing()
}
