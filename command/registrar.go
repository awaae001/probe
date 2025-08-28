package command

import "github.com/bwmarrin/discordgo"

// Command is an interface for application commands.
type Command interface {
	Definition() *discordgo.ApplicationCommand
}

// AllCommands holds all the command instances.
var AllCommands = []Command{
	&ScanCommand{},
	&PingCommand{},
}

// GetCommandDefinitions returns a slice of all command definitions.
func GetCommandDefinitions() []*discordgo.ApplicationCommand {
	defs := make([]*discordgo.ApplicationCommand, len(AllCommands))
	for i, cmd := range AllCommands {
		defs[i] = cmd.Definition()
	}
	return defs
}
