package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	botRouter "github.com/clover0916/gobote/bot_handler/bot_router"
)

type VoteArgs struct {
	Title       string
	Description string
	Choices     []string
	Due         time.Time
	Anonymous   bool
	Mask        bool
	Max         int
	Editable    bool
	Duplicate   bool
}

func VoteCommand() *botRouter.Command {
	return &botRouter.Command{
		Name:        "vote",
		Description: "投票を作成します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "title",
				Description: "Title of the vote",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "choices",
				Description: "Comma-separated choices",
				Required:    true,
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "description",
				Description: "Description of the vote",
				Required:    false,
				Type:        discordgo.ApplicationCommandOptionString,
			},
			{
				Name:        "anonymous",
				Description: "Make votes anonymous",
				Required:    false,
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "mask",
				Description: "Mask vote status",
				Required:    false,
				Type:        discordgo.ApplicationCommandOptionBoolean,
			},
			{
				Name:        "max",
				Description: "Max number of votes per user",
				Required:    false,
				Type:        discordgo.ApplicationCommandOptionInteger,
			},
		},
		Executor: handleVote,
	}
}

func handleVote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	args := parseVoteArgs(options)

	// Create initial vote message
	embed := createVoteEmbed(args, i.Member.User)
	components := createVoteComponents(args)

	// Send the initial vote message
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	}

	err := s.InteractionRespond(i.Interaction, response)
	if err != nil {
		fmt.Printf("Error responding to vote command: %v\n", err)
	}
}

func parseVoteArgs(options []*discordgo.ApplicationCommandInteractionDataOption) VoteArgs {
	args := VoteArgs{
		Due:      time.Now().Add(30 * 24 * time.Hour),
		Max:      1,
		Editable: true,
	}

	for _, opt := range options {
		switch opt.Name {
		case "title":
			args.Title = opt.StringValue()
		case "description":
			args.Description = opt.StringValue()
		case "choices":
			args.Choices = parseChoices(opt.StringValue())
		case "due":
			if due, err := time.Parse(time.RFC3339, opt.StringValue()); err == nil {
				args.Due = due
			}
		case "anonymous":
			args.Anonymous = opt.BoolValue()
		case "mask":
			args.Mask = opt.BoolValue()
		case "max":
			args.Max = int(opt.IntValue())
		case "editable":
			args.Editable = opt.BoolValue()
		case "duplicate":
			args.Duplicate = opt.BoolValue()
		}
	}

	if args.Duplicate {
		args.Editable = false
	}

	return args
}

func parseChoices(choicesStr string) []string {
	// Implement choice parsing logic here
	// For simplicity, we'll just split by comma
	return strings.Split(choicesStr, ",")
}

func createVoteEmbed(args VoteArgs, author *discordgo.User) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       args.Title,
		Description: args.Description,
		Color:       0xFFA500, // Orange color
		Author: &discordgo.MessageEmbedAuthor{
			Name:    author.Username,
			IconURL: author.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: time.Now().Format(time.RFC3339),
		},
	}

	for _, choice := range args.Choices {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   choice,
			Value:  "-",
			Inline: true,
		})
	}

	return embed
}

func createVoteComponents(args VoteArgs) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	var currentRow discordgo.ActionsRow

	for i, choice := range args.Choices {
		if i%5 == 0 && i != 0 {
			components = append(components, currentRow)
			currentRow = discordgo.ActionsRow{}
		}
		button := discordgo.Button{
			Label:    choice,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("choice_%d", i),
		}
		currentRow.Components = append(currentRow.Components, button)
	}
	components = append(components, currentRow)

	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "End/Restart",
				Style:    discordgo.DangerButton,
				CustomID: "toggle",
			},
		},
	})

	return components
}
