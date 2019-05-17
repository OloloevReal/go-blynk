package main

import (
	"flag"
	"log"

	blynk "github.com/OloloevReal/go-blynk"
)

func main() {
	log.Printf("Blynk starting, version %s", blynk.Version)
	defer log.Printf("Go Blynk finished")

	auth := flag.String("auth", "", "set -auth=blynk_token")
	flag.Parse()

	app := blynk.NewBlynk(*auth, "blynk-cloud.com", 80, false)

	if err := app.Connect(); err != nil {
		log.Fatalln(err)
	}
	defer app.Disconnect()

	if err := app.VirtualWrite(9, "7.654"); err != nil {
		log.Println("Send command failed")
	}
	if err := app.VirtualWrite(12, "4.567"); err != nil {
		log.Println("Send command failed")
	}

}
