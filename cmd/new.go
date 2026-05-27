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
	dbType      string
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
			moduleName = fmt.Sprintf("github.com/%s/%s", os.Getenv("USER"), projectName)
		}

		// Handle database type prompting
		if dbType == "" {
			var setupDb string
			fmt.Print("Do you want to set up a database in this project? (y/n): ")
			_, err := fmt.Scanln(&setupDb)
			if err != nil || (setupDb != "y" && setupDb != "yes" && setupDb != "Y" && setupDb != "YES") {
				dbType = "none"
			} else {
				fmt.Println("\nCurrently supported databases:")
				fmt.Println("  1) MongoDB")
				fmt.Println("  2) PostgreSQL")
				fmt.Println("  3) MySQL")
				fmt.Println("  4) DynamoDB")
				for {
					var choice int
					fmt.Print("Select a database (1-4): ")
					_, err := fmt.Scanln(&choice)
					if err != nil || choice < 1 || choice > 4 {
						fmt.Println("Invalid selection. Please enter a number between 1 and 4.")
						// Clear standard input if needed or handle retry
						continue
					}
					switch choice {
					case 1:
						dbType = "mongodb"
					case 2:
						dbType = "postgres"
					case 3:
						dbType = "mysql"
					case 4:
						dbType = "dynamodb"
					}
					break
				}
			}
		} else {
			// Validate flag
			switch dbType {
			case "none", "mongodb", "postgres", "mysql", "dynamodb":
				// valid
			default:
				return fmt.Errorf("invalid database type '%s': must be one of none, mongodb, postgres, mysql, dynamodb", dbType)
			}
		}

		projectPath := filepath.Join(".", projectName)
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}

		gen := generator.NewProjectGenerator(projectPath, projectName, moduleName, dbType)
		if err := gen.Generate(); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}

		fmt.Printf("Successfully created project '%s' at %s (Database: %s)\n", projectName, projectPath, dbType)
		fmt.Println("\nNext steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  go mod tidy")
		fmt.Println("  ginboot build")
		fmt.Println("  ginboot deploy")

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
	newCmd.Flags().StringVar(&dbType, "db", "", "Database type: none, mongodb, postgres, mysql, dynamodb")
}
