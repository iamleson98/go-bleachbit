package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// var name string

var rootCmd = &cobra.Command{
	Use:   "bleachbit",
	Short: "Bleachbit is a garbage cleaner for operatins systems",
}

func init() {

	// rootCmd.Flag().StringVar(&name, "name", "minh", "Just a demo name")
	rootCmd.AddCommand(sayCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
