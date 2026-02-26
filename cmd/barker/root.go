package main

import "github.com/spf13/cobra"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "barker",
	Short: "Monitoring daemon for Linux",
	Long:  "Barker is a single-binary monitoring daemon. It runs checks, sends notifications via Shoutrrr. No database, no UI — just YAML config and a process.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
}
