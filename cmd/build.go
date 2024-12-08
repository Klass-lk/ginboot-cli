package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Ginboot project",
	Long:  `Build the Ginboot project using SAM CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if SAM CLI is installed
		if _, err := exec.LookPath("sam"); err != nil {
			return fmt.Errorf("SAM CLI is not installed. Please install it first: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html")
		}

		// Run sam build
		samCmd := exec.Command("sam", "build")
		samCmd.Stdout = os.Stdout
		samCmd.Stderr = os.Stderr

		if err := samCmd.Run(); err != nil {
			return fmt.Errorf("failed to build project: %w", err)
		}

		fmt.Println("Build completed successfully!")
		return nil
	},
}
