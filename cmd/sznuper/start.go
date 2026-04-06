package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

		// Resolve config path once at startup; reused on every reload.
		cfgPath, err := config.FindPath(cfgFile)
		if err != nil {
			return err
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		applyOptionFlags(cmd, cfg)

		// SIGINT/SIGTERM for graceful shutdown.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		// SIGHUP for config reload.
		sighup := make(chan os.Signal, 1)
		signal.Notify(sighup, syscall.SIGHUP)
		defer signal.Stop(sighup)

		firstStart := true
		for {
			r := runner.New(cfg, logger)
			sched := scheduler.New(r, logger, func(res runner.Result) {
				logResult(logger, res)
			})

			schedCtx, schedCancel := context.WithCancel(ctx)
			schedDone := make(chan struct{})
			go func() {
				sched.Start(schedCtx, cfg.Alerts, scheduler.StartOpts{
					DryRun:        dryRun,
					SkipLifecycle: true,
				})
				close(schedDone)
			}()

			// Fire the appropriate lifecycle event.
			lifecycleAlerts := filterLifecycleAlerts(cfg.Alerts)
			if firstStart {
				logger.Info("sznuper daemon starting", "alerts", len(cfg.Alerts))
				sched.FireLifecycle(schedCtx, lifecycleAlerts, "started", len(cfg.Alerts), dryRun)
				firstStart = false
			} else {
				logger.Info("configuration reloaded", "alerts", len(cfg.Alerts))
				sched.FireLifecycle(schedCtx, lifecycleAlerts, "reload_success", len(cfg.Alerts), dryRun)
			}

			// Wait for shutdown or reload signal.
			reload := false
		waitLoop:
			for {
				select {
				case <-schedDone:
					// SIGINT/SIGTERM propagated via parent ctx.
					stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
					sched.FireLifecycle(stopCtx, lifecycleAlerts, "stopped", len(cfg.Alerts), dryRun)
					stopCancel()
					logger.Info("sznuper daemon stopped")
					schedCancel()
					return nil

				case <-sighup:
					drainSignals(sighup)
					logger.Info("reloading configuration")

					newCfg, loadErr := config.Load(cfgPath)
					if loadErr != nil {
						logger.Error("reload failed: invalid config, keeping current configuration", "error", loadErr)
						sched.FireLifecycle(schedCtx, lifecycleAlerts, "reload_failure", len(cfg.Alerts), dryRun)
						continue
					}
					applyOptionFlags(cmd, newCfg)
					cfg = newCfg
					reload = true
					break waitLoop
				}
			}

			if !reload {
				schedCancel()
				return nil
			}

			// Hard cancel old scheduler, wait for exit.
			schedCancel()
			<-schedDone
		}
	},
}

func init() {
	startCmd.Flags().Bool("dry-run", false, "simulate without sending notifications")
	registerConfigFlags(startCmd)
	rootCmd.AddCommand(startCmd)
}

// drainSignals discards any buffered signals from the channel.
func drainSignals(ch <-chan os.Signal) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// filterLifecycleAlerts returns alerts that have a lifecycle trigger.
func filterLifecycleAlerts(alerts []config.Alert) []config.Alert {
	var out []config.Alert
	for _, a := range alerts {
		if scheduler.HasLifecycleTrigger(a.Triggers) {
			out = append(out, a)
		}
	}
	return out
}

func logResult(logger *slog.Logger, res runner.Result) {
	attrs := []any{
		"alert", res.AlertName,
		"event_type", res.EventType,
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
