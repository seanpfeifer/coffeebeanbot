package main

import (
	"log"

	"github.com/seanpfeifer/coffeebeanbot"
)

func main() {
	defer log.Println("------- BOT SHUTDOWN -------")

	// Load config
	cfg, err := coffeebeanbot.LoadConfigFile("cfg.json")
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	// Start bot
	bot := coffeebeanbot.Bot{Config: *cfg}
	err = bot.Start()
	if err != nil {
		log.Printf("Error starting bot: %v", err)
	}
}
