package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/healthcheck"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config and verify all healthchecks",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(cfgFile)
		if err != nil {
			return err
		}
		applyOptionFlags(cmd, cfg)

		hasError := false
		for _, alert := range cfg.Alerts {
			_, err := healthcheck.Resolve(alert.Healthcheck, healthcheck.ResolveOpts{
				HealthchecksDir: cfg.Options.HealthchecksDir,
				CacheDir:        cfg.Options.CacheDir,
				SHA256:          alert.SHA256,
				ForceVerify:     true,
			})
			if err != nil {
				fmt.Printf("✗ %s (%s): %s\n", alert.Name, alert.Healthcheck, err)
				hasError = true
			} else {
				fmt.Printf("✓ %s (%s)\n", alert.Name, alert.Healthcheck)
			}
		}

		if hasError {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	registerConfigFlags(validateCmd)
	rootCmd.AddCommand(validateCmd)
}
