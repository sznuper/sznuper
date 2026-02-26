package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the barker configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TODO: validate")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
