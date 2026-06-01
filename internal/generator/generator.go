package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type ProjectGenerator struct {
	ProjectPath  string
	ProjectName  string
	ModuleName   string
	GoVersion    string
	DatabaseType string
	StorageType  string
	DeployType   string
}

func NewProjectGenerator(projectPath, projectName, moduleName, goVersion, databaseType, storageType, deployType string) *ProjectGenerator {
	return &ProjectGenerator{
		ProjectPath:  projectPath,
		ProjectName:  projectName,
		ModuleName:   moduleName,
		GoVersion:    goVersion,
		DatabaseType: databaseType,
		StorageType:  storageType,
		DeployType:   deployType,
	}
}

func (g *ProjectGenerator) Generate() error {
	// Create directory structure
	dirs := []string{
		"internal/controller",
		"internal/repository",
		"internal/model",
		"internal/service",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.ProjectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Select templates based on database choice
	var mainTmpl, goModTmpl, dockerComposeTmpl, userModelTmpl, userRepoTmpl string

	switch g.DatabaseType {
	case "mongodb":
		mainTmpl = mainMongoTemplate
		goModTmpl = goModMongoTemplate
		dockerComposeTmpl = dockerComposeMongoTemplate
		userModelTmpl = userModelMongoTemplate
		userRepoTmpl = userRepositoryMongoTemplate
	case "postgres":
		mainTmpl = mainPostgresTemplate
		goModTmpl = goModPostgresTemplate
		dockerComposeTmpl = dockerComposePostgresTemplate
		userModelTmpl = userModelPostgresTemplate
		userRepoTmpl = userRepositoryPostgresTemplate
	case "mysql":
		mainTmpl = mainMysqlTemplate
		goModTmpl = goModMysqlTemplate
		dockerComposeTmpl = dockerComposeMysqlTemplate
		userModelTmpl = userModelMysqlTemplate
		userRepoTmpl = userRepositoryMysqlTemplate
	case "dynamodb":
		mainTmpl = mainDynamodbTemplate
		goModTmpl = goModDynamodbTemplate
		dockerComposeTmpl = dockerComposeDynamodbTemplate
		userModelTmpl = userModelDynamodbTemplate
		userRepoTmpl = userRepositoryDynamodbTemplate
	default: // "none"
		mainTmpl = mainNoneTemplate
		goModTmpl = goModNoneTemplate
		dockerComposeTmpl = dockerComposeNoneTemplate
		userModelTmpl = userModelNoneTemplate
		userRepoTmpl = userRepositoryNoneTemplate
	}

	// Generate files
	files := map[string]string{
		"main.go": mainTmpl,
		"go.mod":  goModTmpl,
	}

	if g.DeployType == "lambda" {
		files["Makefile"] = makefileTemplate
		files["template.yaml"] = templateYamlTemplate
		files["Dockerfile"] = dockerfileTemplate
	}

	if dockerComposeTmpl != "" {
		files["docker-compose.yml"] = dockerComposeTmpl
	}

	for filename, tmpl := range files {
		if err := g.generateFile(filename, tmpl); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filename, err)
		}
	}

	// Generate internal package files
	internalFiles := map[string]string{
		"internal/controller/user_controller.go": userControllerTemplate,
		"internal/model/user.go":                 userModelTmpl,
		"internal/repository/user_repository.go": userRepoTmpl,
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
		GoVersion   string
		HasS3       bool
		HasLambda   bool
	}{
		ProjectName: g.ProjectName,
		ModuleName:  g.ModuleName,
		GoVersion:   g.GoVersion,
		HasS3:       g.StorageType == "s3",
		HasLambda:   g.DeployType == "lambda",
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
