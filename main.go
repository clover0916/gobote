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

	discord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged)
	discord.Token = Token
	if err != nil {
		fmt.Println("Error logging in")
		fmt.Println(err)
	}

	if err = discord.Open(); err != nil {
		fmt.Println(err)
		panic("Error while opening session")
	}

	botRouter.RegisterHandlers(discord)
	discord.AddHandler(commands.HandleVoteInteraction)

	var commandHandlers []*botRouter.Handler
	commandHandler := botRouter.NewCommandHandler(discord, env.GUILD)

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

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc

	fmt.Println("Removing commands...")

	for i := range commandHandlers {
		commands := commandHandlers[i].GetCommands()
		for _, command := range commands {
			err := commandHandlers[i].CommandRemove(command)
			if err != nil {
				panic("error removing command")
			}
		}
	}

	err = discord.Close()
	if err != nil {
		panic("error closing connection")
	}
	fmt.Println("Disconnected")
}
