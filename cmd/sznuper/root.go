package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:     "sznuper",
	Short:   "A lightweight server monitor that runs healthchecks and sends notifications",
	Long:    "A lightweight server monitor that runs healthchecks and sends notifications — Discord, Slack, Telegram, Teams, and more.",
	Version: version,
}

func init() {}

func setupLogger() *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
