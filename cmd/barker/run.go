package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <alert_name>",
	Short: "Run a single alert by name",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Printf("TODO: run (dry-run) alert %q\n", args[0])
			return
		}
		fmt.Printf("TODO: run alert %q\n", args[0])
	},
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "simulate the alert without sending notifications")
	rootCmd.AddCommand(runCmd)
}
