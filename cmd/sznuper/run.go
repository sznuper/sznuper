package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/runner"
)

var runCmd = &cobra.Command{
	Use:   "run [alert_name]",
	Short: "Run alerts once",
	Long:  "Runs a single alert by name, or all alerts if no name is given. Use --dry-run to skip sending notifications.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger := setupLogger()

		cfg, err := config.Resolve(cfgFile)
		if err != nil {
			return err
		}
		applyOptionFlags(cmd, cfg)

		r := runner.New(cfg, logger)
		ctx := context.Background()

		hasError := false
		if len(args) == 1 {
			alert := r.FindAlert(args[0])
			if alert == nil {
				return fmt.Errorf("alert %q not found in config", args[0])
			}
			for res := range r.RunAlert(ctx, alert, dryRun, nil, nil) {
				printResult(res)
				if res.Err != nil {
					hasError = true
				}
			}
		} else {
			for res := range r.RunAll(ctx, dryRun) {
				printResult(res)
				if res.Err != nil {
					hasError = true
				}
			}
		}

		if hasError {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "simulate the alert without sending notifications")
	registerConfigFlags(runCmd)
	rootCmd.AddCommand(runCmd)
}

func printResult(r runner.Result) {
	if r.Err != nil {
		fmt.Printf("✗ Healthcheck: %s\n", r.HealthcheckURI)
		fmt.Printf("  Error (%s): %s\n", r.ErrStage, r.Err)
		if r.Stderr != "" {
			fmt.Printf("  Stderr: %s\n", r.Stderr)
		}
		return
	}

	fmt.Printf("✓ Healthcheck: %s\n", r.HealthcheckURI)
	if len(r.Env) > 0 {
		fmt.Println("  Env:")
		for _, e := range r.Env {
			fmt.Printf("    %s\n", e)
		}
	}
	if len(r.Fields) > 0 {
		fmt.Println("  Fields:")
		for k, v := range r.Fields {
			fmt.Printf("    %s=%s\n", k, v)
		}
	}

	// Show rendered message (pick first if all same, else show per-service).
	if len(r.Rendered) > 0 {
		messages := uniqueMessages(r.Rendered)
		if len(messages) == 1 {
			fmt.Printf("  Rendered: %q\n", messages[0])
		} else {
			for svc, msg := range r.Rendered {
				fmt.Printf("  Rendered (%s): %q\n", svc, msg)
			}
		}
	}

	if len(r.Notified) > 0 {
		label := "Notified"
		if r.DryRun {
			label = "Would notify"
		}
		fmt.Printf("  %s: %s\n", label, strings.Join(r.Notified, ", "))
	}
}

func uniqueMessages(rendered map[string]string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, msg := range rendered {
		if !seen[msg] {
			seen[msg] = true
			unique = append(unique, msg)
		}
	}
	return unique
}
