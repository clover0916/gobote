package main

import (
	"fmt"
	"os"
	"os/signal"

	botRouter "github.com/clover0916/gobote/bot_handler/bot_router"
	"github.com/clover0916/gobote/commands"
	envconfig "github.com/clover0916/gobote/utils"

	"github.com/bwmarrin/discordgo"
)

func main() {
	//Discordのセッションを作成
	env, err := envconfig.NewEnv()
	if err != nil {
		fmt.Println("error loading env")
		env = &envconfig.Env{
			TOKEN: os.Getenv("TOKEN"),
			GUILD: os.Getenv("GUILD"),
		}
	}
	Token := "Bot " + env.TOKEN
	discord, err := discordgo.New(Token)

	// 権限追加
	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged)
	discord.Token = Token
	if err != nil {
		fmt.Println("Error logging in")
		fmt.Println(err)
	}
	// websocketを開いてlistening開始
	if err = discord.Open(); err != nil {
		fmt.Println(err)
		panic("Error while opening session")
	}

	// ハンドラーの登録
	botRouter.RegisterHandlers(discord)

	var commandHandlers []*botRouter.Handler
	// 所属しているサーバすべてにスラッシュコマンドを追加する
	// NewCommandHandlerの第二引数を空にすることで、グローバルでの使用を許可する
	commandHandler := botRouter.NewCommandHandler(discord, env.GUILD)
	// 追加したいコマンドをここに追加
	err = commandHandler.CommandRegister(commands.PingCommand())
	if err != nil {
		panic(fmt.Errorf("error registering Ping command: %v", err))
	}
	err = commandHandler.CommandRegister(commands.HelpCommand())
	if err != nil {
		panic(fmt.Errorf("error registering Help command: %v", err))
	}
	err = commandHandler.CommandRegister(commands.VoteCommand())
	if err != nil {
		panic(fmt.Errorf("error registering Vote command: %v", err))
	}
	commandHandlers = append(commandHandlers, commandHandler)

	fmt.Println("Discordに接続しました。")
	fmt.Println("終了するにはCtrl+Cを押してください。")

	// Ctrl+Cを受け取るためのチャンネル
	sc := make(chan os.Signal, 1)
	// Ctrl+Cを受け取る
	signal.Notify(sc, os.Interrupt)
	<-sc //プログラムが終了しないようロック

	fmt.Println("Removing commands...")

	// コマンドを削除
	for i := range commandHandlers {
		// すべてのコマンドを取得
		commands := commandHandlers[i].GetCommands()
		for _, command := range commands {
			err := commandHandlers[i].CommandRemove(command)
			if err != nil {
				panic("error removing command")
			}
		}
	}

	// websocketを閉じる
	err = discord.Close()
	if err != nil {
		panic("error closing connection")
	}
	fmt.Println("Disconnected")
}
