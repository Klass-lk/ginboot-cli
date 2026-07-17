package cmd

import (
	"fmt"
	"github.com/klass-lk/ginboot-cli/internal/generator"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"regexp"
)

var (
	projectName string
	moduleName  string
	goVersion   string
	dbType      string
	storageType string
	deployType  string
)

var newCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Create a new Ginboot project",
	Long:  `Create a new Ginboot project with a standard directory structure and configuration files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName = args[0]

		// Validate project name (alphanumeric only)
		if !isValidProjectName(projectName) {
			return fmt.Errorf("invalid project name '%s': must contain only letters and numbers", projectName)
		}

		if moduleName == "" {
			user := os.Getenv("USER")
			if user == "" {
				user = "example"
			}
			moduleName = fmt.Sprintf("github.com/%s/%s", user, projectName)
		}

		if goVersion == "" {
			goVersion = "1.21"
		}

		// If any required config is empty, run the Bubble Tea TUI Wizard
		if dbType == "" || storageType == "" || deployType == "" {
			var err error
			dbType, storageType, deployType, err = runWizard()
			if err != nil {
				return err
			}
		} else {
			// Validate flags
			switch dbType {
			case "none", "mongodb", "postgres", "mysql", "dynamodb":
				// valid
			default:
				return fmt.Errorf("invalid database type '%s': must be one of none, mongodb, postgres, mysql, dynamodb", dbType)
			}

			switch storageType {
			case "none", "s3":
				// valid
			default:
				return fmt.Errorf("invalid storage type '%s': must be one of none, s3", storageType)
			}

			switch deployType {
			case "http", "lambda":
				// valid
			default:
				return fmt.Errorf("invalid deployment type '%s': must be one of http, lambda", deployType)
			}
		}

		projectPath := filepath.Join(".", projectName)
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}

		gen := generator.NewProjectGenerator(projectPath, projectName, moduleName, goVersion, dbType, storageType, deployType)
		if err := gen.Generate(); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}

		fmt.Printf("Successfully created project '%s' at %s (Database: %s, Storage: %s, Deploy: %s)\n", projectName, projectPath, dbType, storageType, deployType)
		fmt.Println("\nNext steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  go mod tidy")
		if deployType == "lambda" {
			fmt.Println("  ginboot build")
			fmt.Println("  ginboot deploy")
		} else {
			fmt.Println("  go run main.go")
		}

		return nil
	},
}

// isValidProjectName checks if the project name contains only letters and numbers
func isValidProjectName(name string) bool {
	matched, _ := regexp.MatchString("^[a-zA-Z0-9]+$", name)
	return matched
}

func init() {
	newCmd.Flags().StringVar(&moduleName, "module", "", "Go module name (default: github.com/username/project-name)")
	newCmd.Flags().StringVar(&goVersion, "go-version", "", "Go version (default: 1.21)")
	newCmd.Flags().StringVar(&dbType, "db", "", "Database type: none, mongodb, postgres, mysql, dynamodb")
	newCmd.Flags().StringVar(&storageType, "storage", "", "Storage type: none, s3")
	newCmd.Flags().StringVar(&deployType, "deploy", "", "Deployment type: http, lambda")
}
