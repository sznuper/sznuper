package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the barker daemon",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TODO: start")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
