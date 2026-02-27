package main

import (
	"reflect"
	"strings"

	"github.com/sznuper/sznuper/internal/config"
	"github.com/spf13/cobra"
)

// registerOptionFlags adds a persistent --flag for every field in config.Options,
// deriving the flag name from the yaml struct tag (snake_case → kebab-case).
func registerOptionFlags(cmd *cobra.Command) {
	t := reflect.TypeOf(config.Options{})
	for i := range t.NumField() {
		yamlTag := t.Field(i).Tag.Get("yaml")
		flagName := strings.ReplaceAll(yamlTag, "_", "-")
		cmd.PersistentFlags().String(flagName, "", "override "+yamlTag)
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
