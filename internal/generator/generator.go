package generator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func getLatestGinbootVersion() string {
	resp, err := http.Get("https://api.github.com/repos/Klass-lk/GinBoot/releases/latest")
	if err != nil {
		return "v1.14.2"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "v1.14.2"
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "v1.14.2"
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "v1.14.2"
	}

	return release.TagName
}

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
		"internal/di",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.ProjectPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Select templates based on database choice
	var mainTmpl, goModTmpl, dockerComposeTmpl, userModelTmpl, userRepoTmpl string

	mainTmpl = mainTemplate

	switch g.DatabaseType {
	case "mongodb":
		goModTmpl = goModMongoTemplate
		dockerComposeTmpl = dockerComposeMongoTemplate
		userModelTmpl = userModelMongoTemplate
		userRepoTmpl = userRepositoryMongoTemplate
	case "postgres":
		goModTmpl = goModPostgresTemplate
		dockerComposeTmpl = dockerComposePostgresTemplate
		userModelTmpl = userModelPostgresTemplate
		userRepoTmpl = userRepositoryPostgresTemplate
	case "mysql":
		goModTmpl = goModMysqlTemplate
		dockerComposeTmpl = dockerComposeMysqlTemplate
		userModelTmpl = userModelMysqlTemplate
		userRepoTmpl = userRepositoryMysqlTemplate
	case "dynamodb":
		goModTmpl = goModDynamodbTemplate
		dockerComposeTmpl = dockerComposeDynamodbTemplate
		userModelTmpl = userModelDynamodbTemplate
		userRepoTmpl = userRepositoryDynamodbTemplate
	default: // "none"
		goModTmpl = goModNoneTemplate
		dockerComposeTmpl = dockerComposeNoneTemplate
		userModelTmpl = userModelNoneTemplate
		userRepoTmpl = "" // Removed for inmemory pattern
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
		"internal/service/user_service.go":       userServiceTemplate,
		"internal/di/container.go":               diContainerTemplate,
	}
	
	if userRepoTmpl != "" {
		internalFiles["internal/repository/user_repository.go"] = userRepoTmpl
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
		ProjectName    string
		ModuleName     string
		GoVersion      string
		DatabaseType   string
		GinbootVersion string
		HasS3          bool
		HasLambda      bool
	}{
		ProjectName:    g.ProjectName,
		ModuleName:     g.ModuleName,
		GoVersion:      g.GoVersion,
		DatabaseType:   g.DatabaseType,
		GinbootVersion: getLatestGinbootVersion(),
		HasS3:          g.StorageType == "s3",
		HasLambda:      g.DeployType == "lambda",
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
