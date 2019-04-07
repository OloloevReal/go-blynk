package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	blynk "github.com/OloloevReal/go-blynk"
)

func main() {
	log.Printf("Blynk starting, version %s", blynk.Version)
	defer log.Printf("Go Blynk finished")

	auth := flag.String("auth", "", "set -auth=blynk_token")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Print("[WARN] interrupt signal")
		cancel()
	}()

	//80 - TCP
	//443 - SSL/TCP
	app := blynk.NewBlynk(*auth, "blynk-cloud.com", 80, false)

	go func() {
		<-ctx.Done()
		app.Stop()
	}()

	app.OnReadFunc = func(resp *blynk.BlynkRespose) {
		log.Println("Received:", resp)
	}

	if err := app.Connect(); err != nil {
		log.Fatalln(err)
	}

	if err := app.VirtualWrite(0, "7.654"); err != nil {
		log.Println("Send command failed")
	}
	if err := app.VirtualWrite(1, "4.567"); err != nil {
		log.Println("Send command failed")
	}

	if err := app.VirtualRead(0, 1, 3, 2); err != nil {
		log.Println("Read value failed")
	}

	if err := app.VirtualRead(4); err != nil {
		log.Println("Read value failed")
	}

	app.Processing()
}
