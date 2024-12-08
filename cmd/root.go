package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ginboot",
	Short: "Ginboot CLI - A tool for managing Ginboot projects",
	Long: `Ginboot CLI is a command line tool for creating and managing Ginboot projects.
It helps you scaffold new projects, build and deploy them to AWS Lambda.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(deployCmd)
}
