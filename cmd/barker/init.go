package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new barker configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TODO: init")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
