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

		projectPath := filepath.Join(".", projectName)
		if err := os.MkdirAll(projectPath, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}

		gen := generator.NewProjectGenerator(projectPath, projectName, moduleName)
		if err := gen.Generate(); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}

		fmt.Printf("Successfully created project '%s' at %s\n", projectName, projectPath)
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
}
