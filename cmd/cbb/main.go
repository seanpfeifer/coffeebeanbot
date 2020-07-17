package main

// Copyright 2017 Sean A. Pfeifer

import (
	"flag"

	"github.com/hashicorp/go-hclog"

	"github.com/seanpfeifer/coffeebeanbot"
)

const (
	defaultConfigFile  = "cfg.json"
	defaultSecretsFile = "./secrets/discord.json"
)

func main() {
	logger := createLogger("cbb")
	defer logger.Info("------- BOT SHUTDOWN -------")

	// Parse config path
	configPath := flag.String("cfg", defaultConfigFile, "the config to start the bot with")
	secretsPath := flag.String("secrets", defaultSecretsFile, "the secrets file to load")
	flag.Parse()

	// Load config
	cfg, err := coffeebeanbot.LoadConfigFile(*configPath)
	if coffeebeanbot.LogIfError(logger, err, "Error loading config") {
		return
	}

	// Load secrets
	secrets, err := coffeebeanbot.LoadSecretsFile(*secretsPath)
	if coffeebeanbot.LogIfError(logger, err, "Error loading secrets") {
		return
	}

	// Start bot
	bot := coffeebeanbot.NewBot(*cfg, *secrets, logger)
	err = bot.Start()
	coffeebeanbot.LogIfError(logger, err, "Error starting bot")
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
