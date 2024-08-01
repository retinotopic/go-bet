package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/retinotopic/go-bet/internal/queue"
	"github.com/retinotopic/go-bet/internal/router"
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

	flag.Parse()
	fmt.Println(*addr)
	srv := router.NewRouter(*addr, *addrQueue, config)
	err = srv.Run()
	if err != nil {
		log.Println(err)
	}
}
