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
	CreatorID   string
}

type Votes struct {
	Votes      [][]VoteDetail `json:"votes"`
	LastUpdate time.Time      `json:"lastupdate"`
	IsEnded    bool           `json:"isended"`
}

type VoteDetail struct {
	ID   string    `json:"id"`
	Time time.Time `json:"time"`
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
				Description: "投票のタイトル",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "choices",
				Description: "カンマ区切りの選択肢リスト",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "description",
				Description: "投票の説明",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "due",
				Description: "締め切り日時 (RFC3339形式)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "anonymous",
				Description: "匿名投票",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "mask",
				Description: "投票状況を隠す",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "max",
				Description: "ユーザーあたりの最大投票数",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "editable",
				Description: "投票の編集を許可",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "duplicate",
				Description: "重複投票を許可",
				Required:    false,
			},
		},
		Executor: handleVote,
	}
}

func handleVote(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Interaction.ApplicationCommandData().Name != "vote" {
		return
	}
	if i.Interaction.GuildID != i.GuildID {
		return
	}
	options := i.ApplicationCommandData().Options
	args, err := parseVoteArgs(options)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	embed := createVoteEmbed(args, i.Member.User)
	components := createVoteComponents(args)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})

	if err != nil {
		fmt.Printf("Error sending vote message: %v\n", err)
		return
	}

	message, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		fmt.Printf("Error getting interaction response: %v\n", err)
		return
	}

	args.CreatorID = i.Member.User.ID

	votes := Votes{
		Votes:      make([][]VoteDetail, len(args.Choices)),
		LastUpdate: time.Now(),
		IsEnded:    false,
	}
	voteStorage.Store(message.ID, &VoteData{Args: args, Votes: votes})
}

type VoteData struct {
	Args  VoteArgs
	Votes Votes
}

func parseVoteArgs(options []*discordgo.ApplicationCommandInteractionDataOption) (VoteArgs, error) {
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
			due, err := time.Parse(time.RFC3339, opt.StringValue())
			if err != nil {
				return args, fmt.Errorf("無効な締め切り日時形式です: %v", err)
			}
			args.Due = due
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

	if len(args.Choices) < 2 {
		return args, fmt.Errorf("少なくとも2つの選択肢が必要です")
	}

	if len(args.Choices) > 20 {
		return args, fmt.Errorf("選択肢が多すぎます (最大20)")
	}

	return args, nil
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
		Color:       0x40639a, // Brand color
		Fields:      fields,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    author.Username,
			IconURL: author.AvatarURL(""),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("締め切り: %s", args.Due.Format(time.RFC822)),
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
				Label:    "終了/再開",
				Style:    discordgo.DangerButton,
				CustomID: "toggle",
			},
		},
	})

	return components
}

func HandleVoteInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	messageID := i.Message.ID

	voteDataInterface, ok := voteStorage.Load(messageID)
	if !ok {
		respondWithError(s, i, "投票が見つかりません")
		return
	}
	voteData := voteDataInterface.(*VoteData)

	if data.CustomID == "toggle" {
		if i.Member.User.ID != voteData.Args.CreatorID {
			respondWithError(s, i, "投票作成者のみが終了/再開できます")
			return
		}
		voteData.Votes.IsEnded = !voteData.Votes.IsEnded
		updateVoteMessage(s, i, voteData)
		return
	}

	if voteData.Votes.IsEnded {
		respondWithError(s, i, "この投票は終了しています")
		return
	}

	choiceIndex, err := strconv.Atoi(strings.TrimPrefix(data.CustomID, "choice_"))
	if err != nil {
		respondWithError(s, i, "無効な選択です")
		return
	}

	userID := i.Member.User.ID
	voteDetail := VoteDetail{
		ID:   userID,
		Time: time.Now(),
	}

	cancelled, err := validateVote(voteData, choiceIndex, userID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	if !cancelled {
		voteData.Votes.Votes[choiceIndex] = append(voteData.Votes.Votes[choiceIndex], voteDetail)
	}
	voteData.Votes.LastUpdate = time.Now()

	voteStorage.Store(messageID, voteData)
	updateVoteMessage(s, i, voteData)
}

func validateVote(voteData *VoteData, choiceIndex int, userID string) (bool, error) {
	if choiceIndex < 0 || choiceIndex >= len(voteData.Votes.Votes) {
		return false, fmt.Errorf("無効な選択です")
	}

	// ユーザーがすでに投票しているかチェック
	userVoteCount := 0
	for i, choiceVotes := range voteData.Votes.Votes {
		for j, vote := range choiceVotes {
			if vote.ID == userID {
				userVoteCount++
				if i == choiceIndex {
					// 同じ選択肢に対する投票をキャンセル
					voteData.Votes.Votes[i] = append(voteData.Votes.Votes[i][:j], voteData.Votes.Votes[i][j+1:]...)
					return true, nil // 投票がキャンセルされたことを示す
				}
			}
		}
	}

	// 最大投票数をチェック
	if userVoteCount >= voteData.Args.Max {
		return false, fmt.Errorf("投票数の上限に達しています")
	}

	// Duplicateが有効な場合、同じ選択肢への重複投票を防ぐ
	if voteData.Args.Duplicate {
		for _, vote := range voteData.Votes.Votes[choiceIndex] {
			if vote.ID == userID {
				return false, fmt.Errorf("この選択肢にはすでに投票しています")
			}
		}
	} else {
		// Duplicateが無効な場合、他の選択肢への投票を防ぐ
		if userVoteCount > 0 {
			return false, fmt.Errorf("すでに投票しています")
		}
	}

	return false, nil
}

func updateVoteMessage(s *discordgo.Session, i *discordgo.InteractionCreate, voteData *VoteData) {
	embed := i.Message.Embeds[0]
	totalVotes := 0

	for choiceIndex, choiceVotes := range voteData.Votes.Votes {
		totalVotes += len(choiceVotes)
		var value string
		if voteData.Votes.IsEnded || !voteData.Args.Mask {
			percentage := 0
			if totalVotes > 0 {
				percentage = len(choiceVotes) * 100 / totalVotes
			}
			value = fmt.Sprintf("**%d票, %d%%**", len(choiceVotes), percentage)
			if !voteData.Args.Anonymous {
				voterList := make([]string, len(choiceVotes))
				for i, vote := range choiceVotes {
					voterList[i] = fmt.Sprintf("<@%s>", vote.ID)
				}
				value += "\n" + strings.Join(voterList, ", ")
			}
		} else {
			value = "-"
		}
		embed.Fields[choiceIndex].Value = value
	}

	components := i.Message.Components
	if voteData.Votes.IsEnded {
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

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
	if err != nil {
		fmt.Printf("投票メッセージの更新エラー: %v\n", err)
	}
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, errorMessage string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("エラー: %s", errorMessage),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
