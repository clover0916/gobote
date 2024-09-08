package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	botRouter "github.com/clover0916/gobote/bot_handler/bot_router"
)

func HelpCommand() *botRouter.Command {
	return &botRouter.Command{
		Name:        "help",
		Description: "JVSL Botのヘルプを表示します",
		Options:     []*discordgo.ApplicationCommandOption{},
		Executor:    handleHelp,
	}
}

func handleHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Interaction.ApplicationCommandData().Name != "help" {
		return
	}
	if i.Interaction.GuildID != i.GuildID {
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "JVSL Botのヘルプ",
					Description: "JVSL Botのコマンド一覧です",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "/ping",
							Value:  PingCommand().Description,
							Inline: false,
						},
						{
							Name:   "/help",
							Value:  HelpCommand().Description,
							Inline: false,
						},
						{
							Name:   "/vote",
							Value:  VoteCommand().Description,
							Inline: false,
						},
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("error responding to help command: %v\n", err)
	}
}
