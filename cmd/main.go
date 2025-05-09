package main

import (
	"github.com/retinotopic/go-bet/internal/router"
	"log"
	"os"
)

func main() {
	srv := router.NewRouter("0.0.0.0:" + os.Getenv("APP_PORT"))
	err := srv.Run()
	if err != nil {
		log.Fatal("server run:", err)
	}
}
