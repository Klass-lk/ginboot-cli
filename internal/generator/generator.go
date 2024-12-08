package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type ProjectGenerator struct {
	ProjectPath string
	ProjectName string
	ModuleName  string
}

func NewProjectGenerator(projectPath, projectName, moduleName string) *ProjectGenerator {
	return &ProjectGenerator{
		ProjectPath: projectPath,
		ProjectName: projectName,
		ModuleName:  moduleName,
	}
}

func (g *ProjectGenerator) Generate() error {
	// Create directory structure
	dirs := []string{
		"internal/controller",
		"internal/repository",
		"internal/model",
		"internal/route",
		"internal/service",
		"config",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.ProjectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate files
	files := map[string]string{
		"main.go":       mainTemplate,
		"go.mod":        goModTemplate,
		"Makefile":      makefileTemplate,
		"template.yaml": templateYamlTemplate,
	}

	for filename, tmpl := range files {
		if err := g.generateFile(filename, tmpl); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filename, err)
		}
	}

	// Generate internal package files
	internalFiles := map[string]string{
		"internal/controller/user_controller.go": userControllerTemplate,
		"internal/model/user.go":                 userModelTemplate,
		"internal/repository/user_repository.go": userRepositoryTemplate,
		"internal/service/user_service.go":       userServiceTemplate,
	}

	for filename, tmpl := range internalFiles {
		if err := g.generateFile(filename, tmpl); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filename, err)
		}
	}

	return nil
}

func (g *ProjectGenerator) generateFile(filename, tmplContent string) error {
	filePath := filepath.Join(g.ProjectPath, filename)

	tmpl, err := template.New(filename).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	data := struct {
		ProjectName string
		ModuleName  string
	}{
		ProjectName: g.ProjectName,
		ModuleName:  g.ModuleName,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
