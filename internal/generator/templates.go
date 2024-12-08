package generator

const mainTemplate = `package main

import (
	"github.com/klass-lk/ginboot"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"{{.ModuleName}}/internal/controller"
	"{{.ModuleName}}/internal/repository"
	"github.com/klass-lk/ginboot"
	"log"
	"os"
)

func main() {
	// Initialize MongoDB client
	client, err := mongo.Connect(nil, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(nil)

	database := client.Database(os.Getenv("DB_NAME"))

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)

	// Initialize controllers
	userController := controller.NewUserController(userRepo)

	// Initialize Ginboot app
	app := ginboot.New()

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

const goModTemplate = `module {{ .ModuleName }}

go 1.21

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot v1.0.13
	go.mongodb.org/mongo-driver v1.13.1
)`

const userModelTemplate = `package model

type User struct {
	ID       string ` + "`" + `json:"id"` + "`" + `
	Username string ` + "`" + `json:"username"` + "`" + `
	Email    string ` + "`" + `json:"email"` + "`" + `
}

func (u User) GetID() string {
	return u.ID
}

func (u User) GetCollectionName() string {
	return "users"
}`

const userRepositoryTemplate = `package repository

import (
	"{{ .ModuleName }}/internal/model"
	"github.com/klass-lk/ginboot"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	*ginboot.MongoRepository[model.User]
}

func NewUserRepository(database *mongo.Database) *UserRepository {
	return &UserRepository{
		MongoRepository: ginboot.NewMongoRepository[model.User](database),
	}
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
FROM golang:1.21-alpine AS builder

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

const dockerComposeTemplate = `version: '3.8'

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
    driver: bridge
`
