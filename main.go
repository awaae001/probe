package main

import (
	"discord-bot/bot"
	"discord-bot/handlers"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("Pprof server failed to start: %v", err)
		}
	}()
	bot.Run(handlers.Register)
}
