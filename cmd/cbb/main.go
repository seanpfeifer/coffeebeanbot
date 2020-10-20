package main

import (
	"context"
	"flag"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/hashicorp/go-hclog"
	"go.opencensus.io/examples/exporter"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/stats/view"

	"github.com/seanpfeifer/coffeebeanbot"
	"github.com/seanpfeifer/coffeebeanbot/metrics"
)

const (
	defaultConfigFile  = "cfg.json"
	defaultSecretsFile = "./secrets/discord.json"

	// This should never be less than 60s, or you risk having issues with Stackdriver / a very large bill.
	stackdriverReportingInterval = 60 * time.Second
	stackdriverMetricPrefix      = "cbb"
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

	// Set up metrics
	stopMetrics, err := setupMetricsExporter(cfg.EnableMetrics, cfg.DebugMetrics)
	if coffeebeanbot.LogIfError(logger, err, "Error setting up metrics exporter") {
		return
	}
	defer stopMetrics()

	recorder, err := metrics.NewRecorder()
	if coffeebeanbot.LogIfError(logger, err, "Error creating metrics recorder") {
		return
	}

	// Start bot
	bot := coffeebeanbot.NewBot(*cfg, *secrets, logger, *recorder)
	err = bot.Start()
	coffeebeanbot.LogIfError(logger, err, "Error starting bot")
}

func setupMetricsExporter(enableMetrics, debugMetrics bool) (func(), error) {
	noop := func() {}

	if !enableMetrics {
		return noop, nil
	}

	if debugMetrics {
		// Note that this registration uses the "defaultReportingDuration" of 10s
		view.RegisterExporter(&exporter.PrintExporter{})
		return noop, nil
	} else {
		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			ReportingInterval: stackdriverReportingInterval,
			MetricPrefix:      stackdriverMetricPrefix,
		})
		if err != nil {
			return noop, err
		}
		if err = exporter.StartMetricsExporter(); err != nil {
			return noop, err
		}
		return func() {
			exporter.StopMetricsExporter()
			exporter.Flush()
		}, nil
	}
}

type noopExporter struct{}

func (n *noopExporter) ExportMetrics(ctx context.Context, metrics []*metricdata.Metric) error {
	return nil
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
