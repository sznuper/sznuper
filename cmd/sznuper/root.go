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
	Use:   "sznuper",
	Short: "Monitoring daemon for Linux",
	Long:  "Sznuper is a single-binary monitoring daemon. It runs checks, sends notifications via Shoutrrr. No database, no UI — just YAML config and a process.",
}

func init() {}

func setupLogger() *slog.Logger {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
