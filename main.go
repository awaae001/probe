package main

import (
	"discord-bot/bot"
	"discord-bot/handlers"
)

func main() {
	bot.Run(handlers.Register)
}
