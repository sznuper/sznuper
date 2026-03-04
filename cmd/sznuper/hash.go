package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var hashCmd = &cobra.Command{
	Use:   "hash <file>",
	Short: "Print the sha256 hash of a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		fmt.Printf("%x\n", h.Sum(nil))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hashCmd)
}
