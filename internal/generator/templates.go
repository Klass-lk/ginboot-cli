package generator

const mainTemplate = `package main

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"

	"github.com/klass-lk/ginboot"
	"{{ .ModuleName }}/internal/controller"
	"{{ .ModuleName }}/internal/repository"
	"{{ .ModuleName }}/internal/service"
)

func main() {
	client, err := mongo.Connect(nil, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(nil)

	// Initialize repositories
	userRepository := repository.NewUserRepository(client.Database("{{ .ProjectName }}"))

	// Initialize server
	server := ginboot.New()

	// Set base path for all routes
	server.SetBasePath("/api/v1")

	// Configure CORS with custom settings
	server.CustomCORS(
		[]string{"http://localhost:3000", "https://yourdomain.com"},   // Allow specific origins
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},           // Allow specific methods
		[]string{"Origin", "Content-Type", "Authorization", "Accept"}, // Allow specific headers
		24*time.Hour, // Max age of preflight requests
	)

	// Initialize services
	userService := service.NewUserService(userRepository)

	// Initialize and register controllers
	userController := controller.NewUserController(userService)

	server.RegisterController("/users", userController)

	if err := server.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const goModTemplate = `module {{ .ModuleName }}

go 1.21

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot v1.0.11
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
	"{{ .ModuleName }}/internal/model"
	"{{ .ModuleName }}/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/klass-lk/ginboot"
	"net/http"
)

type UserController struct {
	userService service.UserService
}

func NewUserController(userService service.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

func (c *UserController) Register(group *ginboot.ControllerGroup) {
	group.GET("/:id", c.GetUser)
	group.POST("", c.CreateUser)
}

func (c *UserController) GetUser(ctx *gin.Context) {
	id := ctx.Param("id")

	user, err := c.userService.GetUser(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (c *UserController) CreateUser(ctx *gin.Context) {
	var user model.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.userService.CreateUser(user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, user)
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
