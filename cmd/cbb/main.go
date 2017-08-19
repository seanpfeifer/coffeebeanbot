package main

import (
	"flag"
	"log"

	"github.com/seanpfeifer/coffeebeanbot"
)

const defaultConfigFile = "cfg.json"

func main() {
	defer log.Println("------- BOT SHUTDOWN -------")

	// Parse config path
	configPath := flag.String("cfg", defaultConfigFile, "the config to start the bot with")
	flag.Parse()

	// Load config
	cfg, err := coffeebeanbot.LoadConfigFile(*configPath)
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	// Start bot
	bot := coffeebeanbot.NewBot(*cfg)
	err = bot.Start()
	if err != nil {
		log.Printf("Error starting bot: %v", err)
	}
}
