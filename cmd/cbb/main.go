package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/examples/exporter"
	"go.opencensus.io/stats/view"

	"github.com/seanpfeifer/coffeebeanbot"
	"github.com/seanpfeifer/coffeebeanbot/metrics"
)

const (
	defaultConfigFile  = "cfg.toml"
	defaultSecretsFile = "./secrets/discord.toml"

	// This should never be less than 60s, or you risk having issues with Stackdriver / a very large bill.
	stackdriverReportingInterval = 60 * time.Second
	stackdriverMetricPrefix      = "cbb"
)

type metricsOptions struct {
	StackdriverMetrics bool
	PrintMetrics       bool
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	defer logger.Info("------- BOT SHUTDOWN -------")

	// Parse config + secrets file paths
	configPath := flag.String("cfg", defaultConfigFile, "the config to start the bot with")
	secretsPath := flag.String("secrets", defaultSecretsFile, "the secrets file to load")
	// Also parse metrics options
	var opts metricsOptions
	flag.BoolVar(&opts.StackdriverMetrics, "stackdriver", false, "enables Stackdriver metrics output every "+stackdriverReportingInterval.String())
	flag.BoolVar(&opts.PrintMetrics, "stdoutMetrics", false, "enables printing metrics to stdout every 10s")
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
	stopMetrics, err := setupMetricsExporter(opts)
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

// setupMetricsExporter sets up the OpenCensus metrics exporter, returning a "stopMetrics" func and an error if one occurs.
// Users are expected to call the returned function when the metrics exporter should stop, eg on a "defer" prior to app exit.
func setupMetricsExporter(opts metricsOptions) (func(), error) {
	noop := func() {}

	if opts.PrintMetrics {
		view.RegisterExporter(&exporter.PrintExporter{})
		// No specific stopMetrics func to use here - either return the Stackdriver one or noop at end of func
	}

	if opts.StackdriverMetrics {
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

		// No issues, let's return our stopMetrics func since there are no others to worry about
		return func() {
			exporter.StopMetricsExporter()
			exporter.Flush()
		}, nil
	}

	return noop, nil
}
