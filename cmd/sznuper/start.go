package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/runner"
	"github.com/sznuper/sznuper/internal/scheduler"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the sznuper daemon",
	Long:  "Starts the sznuper daemon, running each alert on its configured interval. Use --dry-run to skip sending notifications.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger := setupLogger()

		cfg, err := config.Resolve(cfgFile)
		if err != nil {
			return err
		}
		applyOptionFlags(cmd, cfg)

		r := runner.New(cfg, logger)

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		sched := scheduler.New(r, logger, func(res runner.Result) {
			logResult(logger, res)
		})

		logger.Info("sznuper daemon starting", "alerts", len(cfg.Alerts))
		sched.Start(ctx, cfg.Alerts, dryRun)
		logger.Info("sznuper daemon stopped")
		return nil
	},
}

func init() {
	startCmd.Flags().Bool("dry-run", false, "simulate without sending notifications")
	registerConfigFlags(startCmd)
	rootCmd.AddCommand(startCmd)
}

func logResult(logger *slog.Logger, res runner.Result) {
	attrs := []any{
		"alert", res.AlertName,
		"status", res.Status,
		"duration", res.Duration,
	}
	switch {
	case res.Err != nil:
		logger.Error("alert failed", append(attrs, "stage", res.ErrStage, "error", res.Err)...)
	case res.Suppressed:
		logger.Info("notification suppressed by cooldown", attrs...)
	case res.IsRecovery:
		logger.Info("recovery notification sent", attrs...)
	default:
		logger.Info("alert completed", attrs...)
	}
}
