package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	botRouter "github.com/clover0916/gobote/bot_handler/bot_router"
)

func PingCommand() *botRouter.Command {
	return &botRouter.Command{
		Name:        "ping",
		Description: "Pong!",
		Options:     []*discordgo.ApplicationCommandOption{},
		Executor:    handlePing,
	}
}

func handlePing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Interaction.ApplicationCommandData().Name != "ping" {
		return
	}
	if i.Interaction.GuildID != i.GuildID {
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pong",
		},
	})
	if err != nil {
		fmt.Printf("error responding to ping command: %v\n", err)
	}
}
