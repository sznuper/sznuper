package main

import (
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sznuper/sznuper/internal/config"
)

// registerConfigFlags registers --config, --verbose, and all options override
// flags on cmd. Call this in init() for commands that load a config file.
func registerConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&cfgFile, "config", "", "config file path")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	t := reflect.TypeOf(config.Options{})
	for i := range t.NumField() {
		yamlTag := t.Field(i).Tag.Get("yaml")
		flagName := strings.ReplaceAll(yamlTag, "_", "-")
		cmd.Flags().String(flagName, "", "override "+yamlTag)
	}
}

// applyOptionFlags overlays CLI flag values onto the config. Only flags
// explicitly set by the user are applied.
func applyOptionFlags(cmd *cobra.Command, cfg *config.Config) {
	t := reflect.TypeOf(cfg.Options)
	v := reflect.ValueOf(&cfg.Options).Elem()
	for i := range t.NumField() {
		yamlTag := t.Field(i).Tag.Get("yaml")
		flagName := strings.ReplaceAll(yamlTag, "_", "-")
		if cmd.Flags().Changed(flagName) {
			val, _ := cmd.Flags().GetString(flagName)
			v.Field(i).SetString(val)
		}
	}
}
