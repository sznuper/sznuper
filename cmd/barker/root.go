package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "barker",
	Short: "Monitoring daemon for Linux",
	Long:  "Barker is a single-binary monitoring daemon. It runs checks, sends notifications via Shoutrrr. No database, no UI — just YAML config and a process.",
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "config file path")
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}
