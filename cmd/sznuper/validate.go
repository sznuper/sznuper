package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sznuper/sznuper/internal/config"
	"github.com/sznuper/sznuper/internal/healthcheck"
	"github.com/sznuper/sznuper/internal/notify"
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

		// Validate channel definitions (dry-run Shoutrrr sender creation).
		for name, ch := range cfg.Channels {
			if hasTemplateVar(ch.URL) {
				fmt.Printf("~ %s (skipped: URL contains template variables)\n", name)
				continue
			}
			t := notify.Target{
				ChannelName: name,
				URL:         ch.URL,
				Params:      ch.Params,
			}
			if err := notify.Validate(t); err != nil {
				fmt.Printf("✗ channel %s: %s\n", name, err)
				hasError = true
			} else {
				fmt.Printf("✓ channel %s\n", name)
			}
		}

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

			for _, bad := range checkChannelRefs(alert, cfg.Channels) {
				fmt.Printf("✗ %s: unknown channel %q\n", alert.Name, bad)
				hasError = true
			}
		}

		if hasError {
			os.Exit(1)
		}
		return nil
	},
}

// checkChannelRefs returns channel names referenced by the alert that are
// not defined in the channels map.
func checkChannelRefs(alert config.Alert, channels map[string]config.Channel) []string {
	var bad []string
	for _, nt := range alert.Notify {
		if _, ok := channels[nt.Channel]; !ok {
			bad = append(bad, nt.Channel)
		}
	}
	if alert.Events != nil {
		for _, ov := range alert.Events.Override {
			for _, nt := range ov.Notify {
				if _, ok := channels[nt.Channel]; !ok {
					bad = append(bad, nt.Channel)
				}
			}
		}
	}
	return bad
}

func hasTemplateVar(s string) bool {
	return strings.Contains(s, "{{")
}

func init() {
	registerConfigFlags(validateCmd)
	rootCmd.AddCommand(validateCmd)
}
