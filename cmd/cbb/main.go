package main

// Copyright 2017 Sean A. Pfeifer

import (
	"flag"

	"github.com/hashicorp/go-hclog"

	"github.com/seanpfeifer/coffeebeanbot"
)

const defaultConfigFile = "cfg.json"

func main() {
	logger := createLogger("cbb")
	defer logger.Info("------- BOT SHUTDOWN -------")

	// Parse config path
	configPath := flag.String("cfg", defaultConfigFile, "the config to start the bot with")
	flag.Parse()

	// Load config
	cfg, err := coffeebeanbot.LoadConfigFile(*configPath)
	if err != nil {
		logger.Error("Error loading config", "error", err)
		return
	}

	// Start bot
	bot := coffeebeanbot.NewBot(*cfg, logger)
	err = bot.Start()
	if err != nil {
		logger.Error("Error starting bot", "error", err)
	}
}

type hclogWrapper struct {
	hclog.Logger
}

func (h *hclogWrapper) Named(name string) coffeebeanbot.Logger {
	return &hclogWrapper{h.Logger.Named(name)}
}

func createLogger(name string) coffeebeanbot.Logger {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  name,
		Level: hclog.Info,
	})

	return &hclogWrapper{logger}
}

// What follows is an example of replacing the logger with Uber's `zap` library.
/*
type zapWrapper struct {
	*zap.SugaredLogger
}

func (z *zapWrapper) Info(msg string, kvPairs ...interface{}) {
	z.SugaredLogger.Infow(msg, kvPairs...)
}

func (z *zapWrapper) Error(msg string, kvPairs ...interface{}) {
	z.SugaredLogger.Errorw(msg, kvPairs...)
}

func (z *zapWrapper) Named(name string) coffeebeanbot.Logger {
	return &zapWrapper{z.SugaredLogger.Named(name)}
}

func createZapLogger(name string) coffeebeanbot.Logger {
	logger, _ := zap.NewProduction()

	return &zapWrapper{logger.Named(name).Sugar()}
}*/
