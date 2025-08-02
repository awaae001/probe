package main

import (
	"discord-bot/bot"
	"discord-bot/handlers"
)

func main() {
	// Convert the slice of command implementations to a slice of bot.Command interfaces.
	var commands []bot.Command
	for _, cmdImpl := range handlers.Commands {
		if cmd, ok := cmdImpl.(bot.Command); ok {
			commands = append(commands, cmd)
		}
	}

	bot.Run(handlers.Register, commands)
}
