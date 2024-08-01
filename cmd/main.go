package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/retinotopic/go-bet/internal/queue"
	"github.com/retinotopic/go-bet/internal/router"
	"github.com/retinotopic/tinyauth/provider"
	"github.com/retinotopic/tinyauth/providers/firebase"
	"github.com/retinotopic/tinyauth/providers/google"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	addrQueue := flag.String("addrQueue", "amqp://guest:guest@localhost:5672/", "rabbitmq address to listen on")
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	var config queue.Config

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	mp := make(map[string]provider.Provider)
	google, err := google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"), "/refresh")
	if err != nil {
		log.Fatalln(err)
	}
	firebase, err := firebase.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"), "refresh")
	if err != nil {
		log.Fatalln(err)
	}

	// with the issuer claim, we know which provider issued token.
	mp[os.Getenv("ISSUER_FIREBASE")] = firebase
	mp[os.Getenv("ISSUER_GOOGLE")] = google
	mp[os.Getenv("ISSUER2_GOOGLE")] = google

	flag.Parse()
	fmt.Println(*addr)
	srv := router.NewRouter(*addr, *addrQueue, config, mp)
	err = srv.Run()
	if err != nil {
		log.Println(err)
	}
}
