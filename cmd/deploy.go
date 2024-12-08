package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type DeployConfig struct {
	StackName        string `yaml:"stack_name"`
	Region           string `yaml:"region"`
	UseDefaultBucket bool   `yaml:"use_default_bucket"`
	S3Bucket         string `yaml:"s3_bucket,omitempty"`
}

// saveConfig saves the deployment configuration to ginboot-app.yml
func saveConfig(config DeployConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile("ginboot-app.yml", data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadConfig loads the deployment configuration from ginboot-app.yml
func loadConfig() (*DeployConfig, error) {
	data, err := os.ReadFile("ginboot-app.yml")
	if err != nil {
		return nil, err
	}

	var config DeployConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// promptUser asks a question and returns the user's answer
func promptUser(question string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", question, defaultValue)
	} else {
		fmt.Printf("%s: ", question)
	}

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	if answer == "" {
		return defaultValue
	}
	return answer
}

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
		// Get current directory name as project name
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName := filepath.Base(currentDir)

		fmt.Printf("🚀 Deploying %s...\n\n", projectName)

		// Check if SAM CLI is installed
		if _, err := exec.LookPath("sam"); err != nil {
			return fmt.Errorf("❌ SAM CLI is not installed. Please install it first: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html")
		}

		var config DeployConfig
		var s3Args []string

		// Check if ginboot-app.yml exists
		if existingConfig, err := loadConfig(); err != nil {
			fmt.Println("📝 No ginboot-app.yml found. Please provide deployment details:")

			// Prompt for required information if not provided via flags
			if stackName == "" {
				stackName = promptUser("Stack name", projectName)
			}
			if region == "" {
				region = promptUser("AWS Region", "us-east-1")
			}

			// Ask about using default S3 bucket
			useDefaultBucket := promptUser("Use default S3 bucket? (Y/n)", "Y")
			useDefault := strings.EqualFold(useDefaultBucket, "y")

			config = DeployConfig{
				StackName:        stackName,
				Region:           region,
				UseDefaultBucket: useDefault,
			}

			if useDefault {
				s3Args = []string{"--resolve-s3", "true"}
				fmt.Println("ℹ️  Using SAM's default S3 bucket")
			} else {
				s3Bucket := promptUser("S3 bucket for deployment artifacts", "")
				if s3Bucket == "" {
					return fmt.Errorf("❌ S3 bucket is required when not using default bucket")
				}
				config.S3Bucket = s3Bucket
				s3Args = []string{"--s3-bucket", s3Bucket}
				fmt.Printf("ℹ️  Using custom S3 bucket: %s\n", s3Bucket)
			}

			// Save configuration for future use
			if err := saveConfig(config); err != nil {
				fmt.Printf("⚠️  Failed to save configuration: %v\n", err)
			} else {
				fmt.Println("💾 Configuration saved to ginboot-app.yml")
			}
		} else {
			fmt.Println("📄 Using existing configuration from ginboot-app.yml")
			config = *existingConfig

			if config.UseDefaultBucket {
				s3Args = []string{"--resolve-s3", "true"}
				fmt.Println("ℹ️  Using SAM's default S3 bucket")
			} else {
				s3Args = []string{"--s3-bucket", config.S3Bucket}
				fmt.Printf("ℹ️  Using custom S3 bucket: %s\n", config.S3Bucket)
			}
		}

		fmt.Println("\n⚙️ Deployment configuration:")
		fmt.Printf("  Stack name: %s\n", config.StackName)
		fmt.Printf("  Region: %s\n", config.Region)
		fmt.Println()

		// Ask for confirmation
		confirm := promptUser("Do you want to proceed with deployment? (y/N)", "N")
		if !strings.EqualFold(confirm, "y") {
			return fmt.Errorf("❌ Deployment cancelled")
		}

		// Prepare deploy command
		deployArgs := []string{
			"deploy",
			"--stack-name", config.StackName,
			"--region", config.Region,
			"--capabilities", "CAPABILITY_IAM",
			"--no-confirm-changeset",
		}
		deployArgs = append(deployArgs, s3Args...)

		// Run sam deploy
		fmt.Println("\n🔨 Starting deployment...")
		var stderr bytes.Buffer
		samCmd := exec.Command("sam", deployArgs...)
		samCmd.Stdout = os.Stdout
		samCmd.Stderr = &stderr

		err = samCmd.Run()
		if err != nil {
			// Check if it's a "no changes" message
			errOutput := stderr.String()
			if strings.Contains(errOutput, "No changes to deploy") {
				fmt.Printf("\n✨ Stack %s is up to date. No changes to deploy.\n", config.StackName)
				return nil
			}
			return fmt.Errorf("❌ Deployment failed: %w", err)
		}

		fmt.Printf("\n✨ Successfully deployed %s!\n", projectName)
		return nil
	},
}

func init() {
	deployCmd.Flags().StringVar(&stackName, "stack-name", "", "AWS CloudFormation stack name")
	deployCmd.Flags().StringVar(&region, "region", "", "AWS Region")
}
