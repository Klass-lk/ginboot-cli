package generator

const userControllerTemplate = `package controller

import (
	"{{.ModuleName}}/internal/model"
	"{{.ModuleName}}/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/klass-lk/ginboot"
)

type UserController struct {
	userRepo *repository.UserRepository
}

func NewUserController(userRepo *repository.UserRepository) *UserController {
	return &UserController{
		userRepo: userRepo,
	}
}

func (c *UserController) Register(group *ginboot.ControllerGroup) {
	group.GET("/:id", c.GetUser)
	group.POST("", c.CreateUser)
}

func (c *UserController) GetUser(ctx *ginboot.Context) {
	id := ctx.Param("id")

	// Example of using auth context
	authCtx, err := ctx.GetAuthContext()
	if err != nil {
		return
	}
	// Use auth context data if needed
	_ = authCtx.UserID

	user, err := c.userRepo.FindById(id)
	if err != nil {
		ctx.JSON(404, gin.H{"error": "User not found"})
		return
	}

	ctx.JSON(200, user)
}

func (c *UserController) CreateUser(ctx *ginboot.Context) {
	var user model.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := c.userRepo.Save(user); err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(201, user)
}`

const userServiceTemplate = `package service

import (
	"{{ .ModuleName }}/internal/model"
	"{{ .ModuleName }}/internal/repository"
)

type UserService interface {
	GetUser(id string) (model.User, error)
	CreateUser(user model.User) error
}

type userService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetUser(id string) (model.User, error) {
	return s.userRepo.FindById(id)
}

func (s *userService) CreateUser(user model.User) error {
	return s.userRepo.Save(user)
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
	"os"
	{{ if .HasS3 }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
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
	github.com/klass-lk/ginboot v1.11.0
	github.com/klass-lk/ginboot/db/mongo v1.11.0
	go.mongodb.org/mongo-driver v1.17.1
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 v1.11.0{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda v1.11.0{{ end }}
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
	"os"
	{{ if .HasS3 }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
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
	"os"
	{{ if .HasS3 }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
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
	github.com/klass-lk/ginboot v1.11.0
	github.com/klass-lk/ginboot/db/sql v1.11.0
	github.com/lib/pq v1.10.9
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 v1.11.0{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda v1.11.0{{ end }}
)`

const goModMysqlTemplate = `module {{ .ModuleName }}

go {{ .GoVersion }}

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot v1.11.0
	github.com/klass-lk/ginboot/db/sql v1.11.0
	github.com/go-sql-driver/mysql v1.8.1
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 v1.11.0{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda v1.11.0{{ end }}
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
	"os"
	{{ if .HasS3 }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
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
	github.com/klass-lk/ginboot v1.11.0
	github.com/klass-lk/ginboot/db/dynamodb v1.11.0
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 v1.11.0{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda v1.11.0{{ end }}
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

const mainNoneTemplate = `package main

import (
	"log"
	{{ if or .HasLambda .HasS3 }}"os"{{ end }}
	{{ if .HasS3 }}"context"{{ end }}

	"github.com/klass-lk/ginboot"
	{{ if .HasLambda }}"github.com/klass-lk/ginboot/runtime/lambda"{{ end }}
	{{ if .HasS3 }}"github.com/klass-lk/ginboot/storage/s3"{{ end }}
	"{{.ModuleName}}/internal/controller"
	"{{.ModuleName}}/internal/repository"
)

func main() {
	// Initialize repositories (In-Memory)
	userRepo := repository.NewUserRepository()

	// Initialize controllers
	userController := controller.NewUserController(userRepo)

	// Initialize Ginboot app
	app := ginboot.New()

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

const goModNoneTemplate = `module {{ .ModuleName }}

go 1.25.0

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot v1.11.0
	{{ if .HasS3 }}github.com/klass-lk/ginboot/storage/s3 v1.11.0{{ end }}
	{{ if .HasLambda }}github.com/klass-lk/ginboot/runtime/lambda v1.11.0{{ end }}
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
	ID       string ` + "`" + `json:"id"` + "`" + `
	Username string ` + "`" + `json:"username"` + "`" + `
	Email    string ` + "`" + `json:"email"` + "`" + `
}`

const userRepositoryNoneTemplate = `package repository

import (
	"errors"
	"sync"
	"{{ .ModuleName }}/internal/model"
)

type UserRepository struct {
	mu    sync.RWMutex
	users map[string]model.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[string]model.User),
	}
}

func (r *UserRepository) FindById(id string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[id]
	if !ok {
		return model.User{}, errors.New("user not found")
	}
	return user, nil
}

func (r *UserRepository) Save(user model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}`
