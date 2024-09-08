package commands

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
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

type Votes struct {
	Votes      [][]VoteDetail
	LastUpdate time.Time
	IsEnded    bool
}

type VoteDetail struct {
	ID   string
	Time time.Time
}

var (
	voteStorage sync.Map
)

func VoteCommand() *botRouter.Command {
	return &botRouter.Command{
		Name:        "vote",
		Description: "投票を作成します",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "title",
				Description: "Vote title",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "choices",
				Description: "Comma-separated list of choices",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "description",
				Description: "Vote description",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "due",
				Description: "Due date (RFC3339 format)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "anonymous",
				Description: "Anonymous vote",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "mask",
				Description: "Mask vote status",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "max",
				Description: "Max votes per user",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "editable",
				Description: "Allow vote editing",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "duplicate",
				Description: "Allow duplicate votes",
				Required:    false,
			},
		},
		Executor: handleVote,
	}
}

func handleVote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	args := parseVoteArgs(options)

	embed := createVoteEmbed(args, i.Member.User)
	components := createVoteComponents(args)

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})

	if err != nil {
		fmt.Printf("Error sending vote message: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred while creating the vote.",
			},
		})
	}
}

func parseVoteArgs(options []*discordgo.ApplicationCommandInteractionDataOption) VoteArgs {
	args := VoteArgs{
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
			args.Choices = strings.Split(opt.StringValue(), ",")
		case "due":
			args.Due, _ = time.Parse(time.RFC3339, opt.StringValue())
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

	if args.Due.IsZero() {
		args.Due = time.Now().Add(30 * 24 * time.Hour)
	}

	if args.Duplicate {
		args.Editable = false
	}

	return args
}

func createVoteEmbed(args VoteArgs, author *discordgo.User) *discordgo.MessageEmbed {
	fields := make([]*discordgo.MessageEmbedField, len(args.Choices))
	for i, choice := range args.Choices {
		fields[i] = &discordgo.MessageEmbedField{
			Name:   choice,
			Value:  "-",
			Inline: true,
		}
	}

	return &discordgo.MessageEmbed{
		Title:       args.Title,
		Description: args.Description,
		Color:       0xFFA500, // Orange
		Fields:      fields,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    author.Username,
			IconURL: author.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: time.Now().Format(time.RFC3339),
		},
	}
}

func createVoteComponents(args VoteArgs) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent

	for i := 0; i < len(args.Choices); i += 5 {
		end := i + 5
		if end > len(args.Choices) {
			end = len(args.Choices)
		}

		row := discordgo.ActionsRow{}
		for j, choice := range args.Choices[i:end] {
			row.Components = append(row.Components, discordgo.Button{
				Label:    choice,
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("choice_%d", i+j),
			})
		}
		components = append(components, row)
	}

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

func handleVoteInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	messageID := i.Message.ID

	votesInterface, ok := voteStorage.Load(messageID)
	if !ok {
		respondWithError(s, i, "Vote not found")
		return
	}
	votes := votesInterface.(Votes)

	if data.CustomID == "toggle" {
		if i.Member.User.ID != i.Message.Author.ID {
			respondWithError(s, i, "Only the vote creator can end/restart the vote")
			return
		}
		votes.IsEnded = !votes.IsEnded
		updateVoteMessage(s, i, votes)
		return
	}

	if votes.IsEnded {
		respondWithError(s, i, "This vote has ended")
		return
	}

	choiceIndex, err := strconv.Atoi(strings.TrimPrefix(data.CustomID, "choice_"))
	if err != nil {
		respondWithError(s, i, "Invalid choice")
		return
	}

	userID := i.Member.User.ID
	voteDetail := VoteDetail{
		ID:   userID,
		Time: time.Now(),
	}

	if err := validateVote(&votes, choiceIndex, userID); err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	votes.Votes[choiceIndex] = append(votes.Votes[choiceIndex], voteDetail)
	votes.LastUpdate = time.Now()

	voteStorage.Store(messageID, votes)
	updateVoteMessage(s, i, votes)
}

func validateVote(votes *Votes, choiceIndex int, userID string) error {
	if choiceIndex < 0 || choiceIndex >= len(votes.Votes) {
		return fmt.Errorf("invalid choice")
	}

	voteCount := 0
	for _, choiceVotes := range votes.Votes {
		for _, vote := range choiceVotes {
			if vote.ID == userID {
				voteCount++
			}
		}
	}

	if voteCount >= 1 {
		return fmt.Errorf("you have already voted")
	}

	return nil
}

func updateVoteMessage(s *discordgo.Session, i *discordgo.InteractionCreate, votes Votes) {
	embed := i.Message.Embeds[0]
	totalVotes := 0

	for choiceIndex, choiceVotes := range votes.Votes {
		totalVotes += len(choiceVotes)
		var value string
		if votes.IsEnded {
			percentage := 0
			if totalVotes > 0 {
				percentage = len(choiceVotes) * 100 / totalVotes
			}
			value = fmt.Sprintf("**%d vote(s), %d%%**", len(choiceVotes), percentage)
		} else {
			value = fmt.Sprintf("**%d vote(s)**", len(choiceVotes))
		}
		embed.Fields[choiceIndex].Value = value
	}

	components := i.Message.Components
	if votes.IsEnded {
		for _, row := range components[:len(components)-1] {
			for _, component := range row.(*discordgo.ActionsRow).Components {
				button := component.(*discordgo.Button)
				button.Disabled = true
			}
		}
	} else {
		for _, row := range components[:len(components)-1] {
			for _, component := range row.(*discordgo.ActionsRow).Components {
				button := component.(*discordgo.Button)
				button.Disabled = false
			}
		}
	}

	_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
	if err != nil {
		fmt.Printf("Error updating vote message: %v", err)
	}
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, errorMessage string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Error: %s", errorMessage),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
