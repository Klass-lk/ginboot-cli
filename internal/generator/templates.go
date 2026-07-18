package generator

const userControllerTemplate = `package controller

import (
	"{{.ModuleName}}/internal/model"
	"{{.ModuleName}}/internal/service"
	"github.com/klass-lk/ginboot"
)

type UserController struct {
	userService service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{
		userService: *userService,
	}
}

func (c *UserController) Register(group *ginboot.ControllerGroup) {
	group.GET("/:id", c.GetUser)
	group.POST("", c.CreateUser)
}

func (c *UserController) GetUser(ctx *ginboot.Context) (model.User, error) {
	//id := ctx.Param("id")

	// Example of using auth context
	authCtx, err := ctx.GetAuthContext()
	if err != nil {
		return model.User{}, err
	}
	// Use auth context data if needed
	_ = authCtx.UserID

	user, err := c.userService.GetUser(authCtx.UserID)
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (c *UserController) CreateUser(ctx *ginboot.Context, request model.User) (model.User, error) {
	user, err := c.userService.CreateUser(request)
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}`

const userServiceTemplate = `package service

import (
	"{{ .ModuleName }}/internal/model"
	{{ if eq .DatabaseType "none" }}
	"github.com/klass-lk/ginboot/db/inmemory"
	{{ else }}
	"{{ .ModuleName }}/internal/repository"
	{{ end }}
)

type UserService interface {
	GetUser(id string) (model.User, error)
	CreateUser(user model.User) (model.User, error)
}

type userService struct {
	{{ if eq .DatabaseType "none" }}
	userRepo *inmemory.InMemoryRepository[model.User]
	{{ else }}
	userRepo *repository.UserRepository
	{{ end }}
}

func NewUserService({{ if eq .DatabaseType "none" }}userRepo *inmemory.InMemoryRepository[model.User]{{ else }}userRepo *repository.UserRepository{{ end }}) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetUser(id string) (model.User, error) {
	return s.userRepo.FindById(id)
}

func (s *userService) CreateUser(user model.User) (model.User, error) {
	err := s.userRepo.Save(user)
	return user, err
}`

const makefileTemplate = `.PHONY: build clean build-{{ .ProjectName }}Function

build: clean
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
	mkdir -p bin/
	zip bin/{{ .ProjectName }}.zip bootstrap
	rm bootstrap

build-{{ .ProjectName }}Function:
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
	mkdir -p $(ARTIFACTS_DIR)
	cp bootstrap $(ARTIFACTS_DIR)/

clean:
	rm -rf bin/
	rm -f bootstrap
	rm -f {{ .ProjectName }}.zip`

const templateYamlTemplate = `AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: >
  {{ .ProjectName }}

Globals:
  Function:
    Timeout: 10
    MemorySize: 128

Resources:
  {{ .ProjectName }}API:
    Type: AWS::Serverless::Api
    Properties:
      StageName: prod

  {{ .ProjectName }}Function:
    Type: AWS::Serverless::Function
    Properties:
      CodeUri: .
      Handler: bootstrap
      Runtime: provided.al2
      Architectures:
        - x86_64
      Events:
        ApiEvents:
          Type: Api
          Properties:
            Path: /{proxy+}
            Method: ANY
            RestApiId: !Ref {{ .ProjectName }}API
      Environment:
        Variables:
          STAGE: prod
    Metadata:
      BuildMethod: makefile

Outputs:
  {{ .ProjectName }}Endpoint:
    Description: API Gateway {{ .ProjectName }} Endpoint
    Value:
      Fn::Sub: https://${{"{"}}{{ .ProjectName }}API}.execute-api.${AWS::Region}.amazonaws.com/prod`

const dockerfileTemplate = `# Build stage
FROM golang:{{ .GoVersion }}-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
`

// =============================================================================
// MongoDB Templates
// =============================================================================

const mainMongoTemplate = `package main

import (
	"log"
	{{ if .HasTelemetry }}"log/slog"{{ end }}
	"os"
	{{ if or .HasS3 .HasTelemetry }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
	{{ if .HasTelemetry }}"github.com/klass-lk/ginboot/telemetry"{{ end }}
	"github.com/klass-lk/ginboot/db/mongo"
	{{ if .HasLambda }}"github.com/klass-lk/ginboot/runtime/lambda"{{ end }}
	{{ if .HasS3 }}"github.com/klass-lk/ginboot/storage/s3"{{ end }}
	"{{.ModuleName}}/internal/controller"
	"{{.ModuleName}}/internal/repository"
)

func main() {
	// Initialize MongoDB config and client
	config := mongo.NewMongoConfig().
		WithHost("localhost", 27017).
		WithDatabase(os.Getenv("DB_NAME"))
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// Initialize controllers
	userController := controller.NewUserController(userRepo)

	// Initialize Ginboot app
	app := ginboot.New()

	{{ if .HasTelemetry }}// Setup Telemetry
	shutdown, err := telemetry.Setup(context.Background(), "{{.ProjectName}}", "1.0.0")
	if err != nil {
		log.Printf("Failed to setup telemetry: %v", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	telemetry.Instrument(app, "{{.ProjectName}}", logger){{ end }}

	{{ if .HasS3 }}
	// Initialize file service (AWS S3)
	fileService := s3.NewS3FileService(
		context.Background(),
		os.Getenv("S3_BUCKET"),
		"./local",
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_REGION"),
		"3600",
	)
	app.BindFileService(fileService)
	{{ end }}

	{{ if .HasLambda }}
	// Initialize Lambda runner if running on AWS Lambda
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		app.SetRunner(lambda.NewRunner())
	}
	{{ end }}

	// API routes
	api := app.Group("/api/v1")
	
	// Public routes
	userGroup := api.Group("/users")
	userController.Register(userGroup)

	// Start server
	if err := app.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const goModMongoTemplate = `module {{ .ModuleName }}

go {{ .GoVersion }}

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot {{ .GinbootVersion }}
{{ if .HasTelemetry }}github.com/klass-lk/ginboot/telemetry {{ .GinbootVersion }}
	{{ end }}	github.com/klass-lk/ginboot/db/mongo {{ .GinbootVersion }}
	go.mongodb.org/mongo-driver v1.17.1
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 {{ .GinbootVersion }}{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda {{ .GinbootVersion }}{{ end }}
)`

const dockerComposeMongoTemplate = `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - MONGODB_URI=mongodb://mongodb:27017
      - DB_NAME={{.ProjectName}}
    depends_on:
      - mongodb
    networks:
      - {{.ProjectName}}-network

  mongodb:
    image: mongo:latest
    ports:
      - "27017:27017"
    volumes:
      - mongodb_data:/data/db
    networks:
      - {{.ProjectName}}-network

volumes:
  mongodb_data:

networks:
  {{.ProjectName}}-network:
    driver: bridge`

const userModelMongoTemplate = `package model

type User struct {
	ID       string ` + "`" + `json:"id" bson:"_id" ginboot:"id"` + "`" + `
	Username string ` + "`" + `json:"username" bson:"username"` + "`" + `
	Email    string ` + "`" + `json:"email" bson:"email"` + "`" + `
}

func (u User) GetID() string {
	return u.ID
}

func (u User) GetCollectionName() string {
	return "users"
}`

const userRepositoryMongoTemplate = `package repository

import (
	"{{ .ModuleName }}/internal/model"
	"github.com/klass-lk/ginboot/db/mongo"
	mongoDriver "go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	*mongo.MongoRepository[model.User]
}

func NewUserRepository(database *mongoDriver.Database) *UserRepository {
	return &UserRepository{
		MongoRepository: mongo.NewMongoRepository[model.User](database, "users"),
	}
}`

// =============================================================================
// SQL Templates (PostgreSQL & MySQL)
// =============================================================================

const mainPostgresTemplate = `package main

import (
	"log"
	{{ if .HasTelemetry }}"log/slog"{{ end }}
	"os"
	{{ if or .HasS3 .HasTelemetry }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
	{{ if .HasTelemetry }}"github.com/klass-lk/ginboot/telemetry"{{ end }}
	"github.com/klass-lk/ginboot/db/sql"
	{{ if .HasLambda }}"github.com/klass-lk/ginboot/runtime/lambda"{{ end }}
	{{ if .HasS3 }}"github.com/klass-lk/ginboot/storage/s3"{{ end }}
	"{{.ModuleName}}/internal/controller"
	"{{.ModuleName}}/internal/repository"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize SQL config and client
	config := sql.NewSQLConfig().
		WithDriver("postgres").
		WithHost("localhost", 5432).
		WithCredentials("postgres", "postgres").
		WithDatabase(os.Getenv("DB_NAME"))
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// Initialize controllers
	userController := controller.NewUserController(userRepo)

	// Initialize Ginboot app
	app := ginboot.New()

	{{ if .HasTelemetry }}// Setup Telemetry
	shutdown, err := telemetry.Setup(context.Background(), "{{.ProjectName}}", "1.0.0")
	if err != nil {
		log.Printf("Failed to setup telemetry: %v", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	telemetry.Instrument(app, "{{.ProjectName}}", logger){{ end }}

	{{ if .HasS3 }}
	// Initialize file service (AWS S3)
	fileService := s3.NewS3FileService(
		context.Background(),
		os.Getenv("S3_BUCKET"),
		"./local",
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_REGION"),
		"3600",
	)
	app.BindFileService(fileService)
	{{ end }}

	{{ if .HasLambda }}
	// Initialize Lambda runner if running on AWS Lambda
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		app.SetRunner(lambda.NewRunner())
	}
	{{ end }}

	// API routes
	api := app.Group("/api/v1")
	
	// Public routes
	userGroup := api.Group("/users")
	userController.Register(userGroup)

	// Start server
	if err := app.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const mainMysqlTemplate = `package main

import (
	"log"
	{{ if .HasTelemetry }}"log/slog"{{ end }}
	"os"
	{{ if or .HasS3 .HasTelemetry }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
	{{ if .HasTelemetry }}"github.com/klass-lk/ginboot/telemetry"{{ end }}
	"github.com/klass-lk/ginboot/db/sql"
	{{ if .HasLambda }}"github.com/klass-lk/ginboot/runtime/lambda"{{ end }}
	{{ if .HasS3 }}"github.com/klass-lk/ginboot/storage/s3"{{ end }}
	"{{.ModuleName}}/internal/controller"
	"{{.ModuleName}}/internal/repository"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Initialize SQL config and client
	config := sql.NewSQLConfig().
		WithDriver("mysql").
		WithHost("localhost", 3306).
		WithCredentials("root", "root").
		WithDatabase(os.Getenv("DB_NAME"))
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// Initialize controllers
	userController := controller.NewUserController(userRepo)

	// Initialize Ginboot app
	app := ginboot.New()

	{{ if .HasTelemetry }}// Setup Telemetry
	shutdown, err := telemetry.Setup(context.Background(), "{{.ProjectName}}", "1.0.0")
	if err != nil {
		log.Printf("Failed to setup telemetry: %v", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	telemetry.Instrument(app, "{{.ProjectName}}", logger){{ end }}

	{{ if .HasS3 }}
	// Initialize file service (AWS S3)
	fileService := s3.NewS3FileService(
		context.Background(),
		os.Getenv("S3_BUCKET"),
		"./local",
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_REGION"),
		"3600",
	)
	app.BindFileService(fileService)
	{{ end }}

	{{ if .HasLambda }}
	// Initialize Lambda runner if running on AWS Lambda
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		app.SetRunner(lambda.NewRunner())
	}
	{{ end }}

	// API routes
	api := app.Group("/api/v1")
	
	// Public routes
	userGroup := api.Group("/users")
	userController.Register(userGroup)

	// Start server
	if err := app.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const goModPostgresTemplate = `module {{ .ModuleName }}

go {{ .GoVersion }}

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot {{ .GinbootVersion }}
{{ if .HasTelemetry }}github.com/klass-lk/ginboot/telemetry {{ .GinbootVersion }}
	{{ end }}	github.com/klass-lk/ginboot/db/sql {{ .GinbootVersion }}
	github.com/lib/pq v1.10.9
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 {{ .GinbootVersion }}{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda {{ .GinbootVersion }}{{ end }}
)`

const goModMysqlTemplate = `module {{ .ModuleName }}

go {{ .GoVersion }}

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot {{ .GinbootVersion }}
{{ if .HasTelemetry }}github.com/klass-lk/ginboot/telemetry {{ .GinbootVersion }}
	{{ end }}	github.com/klass-lk/ginboot/db/sql {{ .GinbootVersion }}
	github.com/go-sql-driver/mysql v1.8.1
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 {{ .GinbootVersion }}{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda {{ .GinbootVersion }}{{ end }}
)`

const dockerComposePostgresTemplate = `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_NAME={{.ProjectName}}
    depends_on:
      - postgres
    networks:
      - {{.ProjectName}}-network

  postgres:
    image: postgres:13-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_DB={{.ProjectName}}
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - {{.ProjectName}}-network

volumes:
  postgres_data:

networks:
  {{.ProjectName}}-network:
    driver: bridge`

const dockerComposeMysqlTemplate = `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_NAME={{.ProjectName}}
    depends_on:
      - mysql
    networks:
      - {{.ProjectName}}-network

  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
    environment:
      - MYSQL_DATABASE={{.ProjectName}}
      - MYSQL_ROOT_PASSWORD=root
    volumes:
      - mysql_data:/var/lib/mysql
    networks:
      - {{.ProjectName}}-network

volumes:
  mysql_data:

networks:
  {{.ProjectName}}-network:
    driver: bridge`

const userModelPostgresTemplate = `package model

type User struct {
	ID       string ` + "`" + `json:"id" db:"id" ginboot:"id"` + "`" + `
	Username string ` + "`" + `json:"username" db:"username"` + "`" + `
	Email    string ` + "`" + `json:"email" db:"email"` + "`" + `
}

func (u User) GetTableName() string {
	return "users"
}`

const userModelMysqlTemplate = userModelPostgresTemplate

const userRepositoryPostgresTemplate = `package repository

import (
	"database/sql"
	"{{ .ModuleName }}/internal/model"
	dbSql "github.com/klass-lk/ginboot/db/sql"
)

type UserRepository struct {
	*dbSql.SQLRepository[model.User]
}

func NewUserRepository(db *sql.DB) *UserRepository {
	repo := &UserRepository{
		SQLRepository: dbSql.NewSQLRepository[model.User](db),
	}
	_ = repo.CreateTable()
	return repo
}`

const userRepositoryMysqlTemplate = userRepositoryPostgresTemplate

// =============================================================================
// DynamoDB Templates
// =============================================================================

const mainDynamodbTemplate = `package main

import (
	"log"
	{{ if .HasTelemetry }}"log/slog"{{ end }}
	"os"
	{{ if or .HasS3 .HasTelemetry }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
	{{ if .HasTelemetry }}"github.com/klass-lk/ginboot/telemetry"{{ end }}
	"github.com/klass-lk/ginboot/db/dynamodb"
	{{ if .HasLambda }}"github.com/klass-lk/ginboot/runtime/lambda"{{ end }}
	{{ if .HasS3 }}"github.com/klass-lk/ginboot/storage/s3"{{ end }}
	"{{.ModuleName}}/internal/controller"
	"{{.ModuleName}}/internal/repository"
)

func main() {
	// Initialize DynamoDB Config
	dynamodb.NewDynamoDBConfig().
		WithTableName("{{.ProjectName}}-table").
		WithSkipTableCreation(false)

	// Initialize DynamoDB Client
	client, err := dynamodb.NewDynamoDBClient("us-east-1")
	if err != nil {
		log.Fatal(err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(client)

	// Initialize controllers
	userController := controller.NewUserController(userRepo)

	// Initialize Ginboot app
	app := ginboot.New()

	{{ if .HasTelemetry }}// Setup Telemetry
	shutdown, err := telemetry.Setup(context.Background(), "{{.ProjectName}}", "1.0.0")
	if err != nil {
		log.Printf("Failed to setup telemetry: %v", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	telemetry.Instrument(app, "{{.ProjectName}}", logger){{ end }}

	{{ if .HasS3 }}
	// Initialize file service (AWS S3)
	fileService := s3.NewS3FileService(
		context.Background(),
		os.Getenv("S3_BUCKET"),
		"./local",
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_REGION"),
		"3600",
	)
	app.BindFileService(fileService)
	{{ end }}

	{{ if .HasLambda }}
	// Initialize Lambda runner if running on AWS Lambda
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		app.SetRunner(lambda.NewRunner())
	}
	{{ end }}

	// API routes
	api := app.Group("/api/v1")
	
	// Public routes
	userGroup := api.Group("/users")
	userController.Register(userGroup)

	// Start server
	if err := app.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const goModDynamodbTemplate = `module {{ .ModuleName }}

go {{ .GoVersion }}

require (
	github.com/aws/aws-sdk-go-v2 v1.40.1
	github.com/aws/aws-sdk-go-v2/config v1.28.5
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.50.3
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot {{ .GinbootVersion }}
{{ if .HasTelemetry }}github.com/klass-lk/ginboot/telemetry {{ .GinbootVersion }}
	{{ end }}	github.com/klass-lk/ginboot/db/dynamodb {{ .GinbootVersion }}
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 {{ .GinbootVersion }}{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda {{ .GinbootVersion }}{{ end }}
)`

const dockerComposeDynamodbTemplate = `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - AWS_ACCESS_KEY_ID=dummy
      - AWS_SECRET_ACCESS_KEY=dummy
      - AWS_REGION=us-east-1
    depends_on:
      - dynamodb-local
    networks:
      - {{.ProjectName}}-network

  dynamodb-local:
    image: amazon/dynamodb-local:latest
    ports:
      - "8000:8000"
    command: "-jar DynamoDBLocal.jar -sharedDb -dbPath ."
    volumes:
      - dynamodb_data:/home/dynamodblocal/data
    networks:
      - {{.ProjectName}}-network

volumes:
  dynamodb_data:

networks:
  {{.ProjectName}}-network:
    driver: bridge`

const userModelDynamodbTemplate = `package model

type User struct {
	ID       string ` + "`" + `json:"id" ginboot:"id" dynamodbav:"id"` + "`" + `
	Username string ` + "`" + `json:"username" dynamodbav:"username"` + "`" + `
	Email    string ` + "`" + `json:"email" dynamodbav:"email"` + "`" + `
}`

const userRepositoryDynamodbTemplate = `package repository

import (
	"{{ .ModuleName }}/internal/model"
	"github.com/klass-lk/ginboot/db/dynamodb"
)

type UserRepository struct {
	*dynamodb.DynamoDBRepository[model.User]
}

func NewUserRepository(client dynamodb.DynamoDBAPI) *UserRepository {
	return &UserRepository{
		DynamoDBRepository: dynamodb.NewDynamoDBRepository[model.User](client),
	}
}

func (r *UserRepository) FindById(id string) (model.User, error) {
	return r.DynamoDBRepository.FindById(id, "USER")
}

func (r *UserRepository) Save(user model.User) error {
	return r.DynamoDBRepository.Save(user, "USER")
}`

// =============================================================================
// None (In-Memory) Templates
// =============================================================================

const mainTemplate = `package main

import (
	"log"
	{{ if .HasTelemetry }}"log/slog"{{ end }}
	{{ if or .HasS3 .HasLambda .HasTelemetry }}"os"{{ end }}
	{{ if or .HasS3 .HasTelemetry }}"context"{{ end }}

	"{{.ModuleName}}/internal/di"
	"github.com/klass-lk/ginboot"
	{{ if .HasTelemetry }}"github.com/klass-lk/ginboot/telemetry"{{ end }}
	{{ if .HasLambda }}"github.com/klass-lk/ginboot/runtime/lambda"{{ end }}
	{{ if .HasS3 }}"github.com/klass-lk/ginboot/storage/s3"{{ end }}
)

func main() {
	// Initialize Ginboot app
	app := ginboot.New()

	{{ if .HasTelemetry }}// Setup Telemetry
	shutdown, err := telemetry.Setup(context.Background(), "{{.ProjectName}}", "1.0.0")
	if err != nil {
		log.Printf("Failed to setup telemetry: %v", err)
	}
	defer func() {
		if shutdown != nil {
			_ = shutdown(context.Background())
		}
	}()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	telemetry.Instrument(app, "{{.ProjectName}}", logger){{ end }}

	{{ if .HasS3 }}
	// Initialize file service (AWS S3)
	fileService := s3.NewS3FileService(
		context.Background(),
		os.Getenv("S3_BUCKET"),
		"./local",
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		os.Getenv("AWS_REGION"),
		"3600",
	)
	app.BindFileService(fileService)
	{{ end }}

	{{ if .HasLambda }}
	// Initialize Lambda runner if running on AWS Lambda
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		app.SetRunner(lambda.NewRunner())
	}
	{{ end }}

	app.SetBasePath("/api")
	di.NewContainer(app)

	// Start server
	if err := app.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const goModNoneTemplate = `module {{ .ModuleName }}

go 1.25.0

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/klass-lk/ginboot {{ .GinbootVersion }}
{{ if .HasTelemetry }}github.com/klass-lk/ginboot/telemetry {{ .GinbootVersion }}
	{{ end }}	github.com/klass-lk/ginboot/db/inmemory {{ .GinbootVersion }}
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 {{ .GinbootVersion }}{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda {{ .GinbootVersion }}{{ end }}
)`

const dockerComposeNoneTemplate = `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
`

const userModelNoneTemplate = `package model

type User struct {
	ID       string ` + "`" + `json:"id" ginboot:"id"` + "`" + `
	Username string ` + "`" + `json:"username"` + "`" + `
	Email    string ` + "`" + `json:"email"` + "`" + `
}`

const diContainerTemplate = `package di

import (
	{{ if ne .DatabaseType "none" }}"log"{{ end }}
	{{ if ne .DatabaseType "none" }}"os"{{ end }}

	"{{.ModuleName}}/internal/controller"
	{{ if eq .DatabaseType "none" }}"{{.ModuleName}}/internal/model"{{ end }}
	"{{.ModuleName}}/internal/service"
	{{ if ne .DatabaseType "none" }}"{{.ModuleName}}/internal/repository"{{ end }}
	"github.com/klass-lk/ginboot"
	{{ if eq .DatabaseType "none" }}"github.com/klass-lk/ginboot/db/inmemory"{{ end }}
	{{ if eq .DatabaseType "dynamodb" }}"github.com/klass-lk/ginboot/db/dynamodb"{{ end }}
	{{ if eq .DatabaseType "mongodb" }}"github.com/klass-lk/ginboot/db/mongo"{{ end }}
	{{ if eq .DatabaseType "postgres" }}"github.com/klass-lk/ginboot/db/sql"{{ end }}
	{{ if eq .DatabaseType "mysql" }}"github.com/klass-lk/ginboot/db/sql"{{ end }}
)

type Container struct {
	Services Services
}

type Services struct {
	UserService service.UserService
}

type Repository struct {
	{{ if eq .DatabaseType "none" }}
	UserRepository *inmemory.InMemoryRepository[model.User]
	{{ else }}
	UserRepository *repository.UserRepository
	{{ end }}
}

func NewContainer(engine *ginboot.Server) {
	repos := InitializeRepositories()
	services := InitializeServices(repos)
	InitializeControllers(services, engine)
}

func InitializeRepositories() *Repository {
	{{ if eq .DatabaseType "none" }}
	userRepository := inmemory.NewInMemoryRepository[model.User]()
	return &Repository{
		UserRepository: userRepository,
	}
	{{ else if eq .DatabaseType "mongodb" }}
	config := mongo.NewMongoConfig().
		WithHost("localhost", 27017).
		WithDatabase(os.Getenv("DB_NAME"))
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err)
	}
	userRepository := repository.NewUserRepository(db)
	return &Repository{
		UserRepository: userRepository,
	}
	{{ else if eq .DatabaseType "postgres" }}
	config := sql.NewSQLConfig().
		WithDriver("postgres").
		WithHost("localhost", 5432).
		WithCredentials(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")).
		WithDatabase(os.Getenv("DB_NAME"))
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err)
	}
	userRepository := repository.NewUserRepository(db)
	return &Repository{
		UserRepository: userRepository,
	}
	{{ else if eq .DatabaseType "mysql" }}
	config := sql.NewSQLConfig().
		WithDriver("mysql").
		WithHost("localhost", 3306).
		WithCredentials(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")).
		WithDatabase(os.Getenv("DB_NAME"))
	db, err := config.Connect()
	if err != nil {
		log.Fatal(err)
	}
	userRepository := repository.NewUserRepository(db)
	return &Repository{
		UserRepository: userRepository,
	}
	{{ else if eq .DatabaseType "dynamodb" }}
	client, err := dynamodb.NewDynamoDBClient(os.Getenv("AWS_REGION"))
	if err != nil {
		log.Fatal(err)
	}
	userRepository := repository.NewUserRepository(client)
	return &Repository{
		UserRepository: userRepository,
	}
	{{ end }}
}

func InitializeServices(repos *Repository) *Services {
	userService := service.NewUserService(repos.UserRepository)
	return &Services{
		UserService: userService,
	}
}

func InitializeControllers(services *Services, engine *ginboot.Server) {
	userController := controller.NewUserController(&services.UserService)
	engine.RegisterController("users", userController)
}`
