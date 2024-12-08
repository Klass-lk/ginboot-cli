package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	stackName    string
	region       string
	capabilities []string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the Ginboot project",
	Long:  `Deploy the Ginboot project to AWS using SAM CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if SAM CLI is installed
		if _, err := exec.LookPath("sam"); err != nil {
			return fmt.Errorf("SAM CLI is not installed. Please install it first: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html")
		}

		// Prepare deploy command
		deployArgs := []string{
			"deploy",
			"--stack-name", stackName,
			"--region", region,
			"--capabilities", "CAPABILITY_IAM",
			"--no-confirm-changeset",
		}

		// Run sam deploy
		samCmd := exec.Command("sam", deployArgs...)
		samCmd.Stdout = os.Stdout
		samCmd.Stderr = os.Stderr

		if err := samCmd.Run(); err != nil {
			return fmt.Errorf("failed to deploy project: %w", err)
		}

		fmt.Println("Deployment completed successfully!")
		return nil
	},
}

func init() {
	deployCmd.Flags().StringVar(&stackName, "stack-name", "", "AWS CloudFormation stack name (required)")
	deployCmd.Flags().StringVar(&region, "region", "us-east-1", "AWS region")
	deployCmd.MarkFlagRequired("stack-name")
}
