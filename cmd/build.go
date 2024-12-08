package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Ginboot project",
	Long:  `Build the Ginboot project using SAM CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current directory name as project name
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName := filepath.Base(currentDir)

		fmt.Printf("🚀 Building %s...\n", projectName)

		// Check if SAM CLI is installed
		if _, err := exec.LookPath("sam"); err != nil {
			return fmt.Errorf("❌ SAM CLI is not installed. Please install it first: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html")
		}

		// Run sam build with output captured
		var stdout, stderr bytes.Buffer
		samCmd := exec.Command("sam", "build")
		samCmd.Stdout = &stdout
		samCmd.Stderr = &stderr

		if err := samCmd.Run(); err != nil {
			// If build fails, show SAM's error output
			fmt.Print(stderr.String())
			return fmt.Errorf("❌ Build failed: %w", err)
		}

		// Check if the build artifacts exist
		if _, err := os.Stat(".aws-sam/build"); err != nil {
			return fmt.Errorf("❌ Build failed: build directory not created")
		}

		fmt.Printf("✨ Successfully built %s!\n", projectName)
		fmt.Println("🔍 Build artifacts are available in .aws-sam/build/")
		fmt.Println("\n📝 Next steps:")
		fmt.Println("  ginboot deploy")

		return nil
	},
}
