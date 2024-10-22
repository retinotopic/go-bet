package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	json "github.com/bytedance/sonic"

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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	google, err := google.New(ctx, os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"), "/refresh")
	if err != nil {
		log.Fatalln(err)
	}
	firebase, err := firebase.New(ctx, os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), os.Getenv("REDIRECT"), "refresh")
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
