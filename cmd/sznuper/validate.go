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

			for _, bad := range checkServiceRefs(alert, cfg.Services) {
				fmt.Printf("✗ %s: unknown service %q\n", alert.Name, bad)
				hasError = true
			}
		}

		if hasError {
			os.Exit(1)
		}
		return nil
	},
}

// checkServiceRefs returns service names referenced by the alert that are
// not defined in the services map.
func checkServiceRefs(alert config.Alert, services map[string]config.Service) []string {
	var bad []string
	for _, nt := range alert.Notify {
		if _, ok := services[nt.Service]; !ok {
			bad = append(bad, nt.Service)
		}
	}
	if alert.Events != nil {
		for _, ov := range alert.Events.Override {
			for _, nt := range ov.Notify {
				if _, ok := services[nt.Service]; !ok {
					bad = append(bad, nt.Service)
				}
			}
		}
	}
	return bad
}

func init() {
	registerConfigFlags(validateCmd)
	rootCmd.AddCommand(validateCmd)
}
