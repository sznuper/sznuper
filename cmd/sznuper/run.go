package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/runner"
	"github.com/spf13/cobra"
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

		var results []runner.Result
		if len(args) == 1 {
			alert := r.FindAlert(args[0])
			if alert == nil {
				return fmt.Errorf("alert %q not found in config", args[0])
			}
			results = append(results, r.RunAlert(ctx, alert, dryRun))
		} else {
			results = r.RunAll(ctx, dryRun)
		}

		hasError := false
		for _, res := range results {
			printResult(res)
			if res.Err != nil {
				hasError = true
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
	rootCmd.AddCommand(runCmd)
}

func printResult(r runner.Result) {
	if r.Err != nil {
		fmt.Printf("✗ Check: %s\n", r.CheckURI)
		fmt.Printf("  Error (%s): %s\n", r.ErrStage, r.Err)
		if r.Stderr != "" {
			fmt.Printf("  Stderr: %s\n", r.Stderr)
		}
		return
	}

	fmt.Printf("✓ Check: %s\n", r.CheckURI)
	if len(r.Output) > 0 {
		fmt.Println("  Output:")
		for _, line := range r.Output {
			fmt.Printf("    %s\n", line)
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
