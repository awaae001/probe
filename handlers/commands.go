package handlers

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

// PingCommand is a simple command that replies with "Pong!".
type PingCommand struct{}

func (p *PingCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping the bot",
	}
}

func (p *PingCommand) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong!",
		},
	})
}

func (p *PingCommand) MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	prefix := viper.GetString("bot.prefix")
	if prefix == "" {
		prefix = "!" // Default prefix
	}

	if strings.HasPrefix(m.Content, prefix+p.Definition().Name) {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
}

// Commands is a slice of all available commands.
var Commands = []interface{}{
	&PingCommand{},
}
