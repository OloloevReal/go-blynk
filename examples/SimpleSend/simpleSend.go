package main

import (
	"flag"

	blynk "github.com/OloloevReal/go-blynk"
	slog "github.com/OloloevReal/go-simple-log"
)

func main() {
	slog.Printf("Blynk starting, version %s", blynk.Version)
	defer slog.Println("Go Blynk finished")

	auth := flag.String("auth", "", "set -auth=blynk_token")
	email := flag.String("email", "", "set -email=someemail@gmail.com")
	_ = email
	flag.Parse()

	app := blynk.NewBlynk(*auth)
	app.SetUseSSL(false)
	//app.SetServer("blynk-cloud.com", 80, false)
	app.DisableLogo(false)
	app.SetDebug()

	if err := app.Connect(); err != nil {
		slog.Fatalln(err)
	}
	defer app.Disconnect()

	if err := app.VirtualWrite(9, "7.654"); err != nil {
		slog.Printf("[ERROR] Send command failed")
	}
	if err := app.VirtualWrite(12, "4.567"); err != nil {
		slog.Printf("[ERROR] Send command failed")
	}

}
