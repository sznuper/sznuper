package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var hashCmd = &cobra.Command{
	Use:   "hash <file>",
	Short: "Print the sha256 hash of a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("TODO: hash %q\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(hashCmd)
}
