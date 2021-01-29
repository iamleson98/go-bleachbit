package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var name string

func init() {
	sayCmd.Flags().StringVar(&name, "name", "minh", "Just a name")
}

var sayCmd = &cobra.Command{
	Use:   "say",
	Short: "Say something to people",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(name)
		return nil
	},
}
