package generator

const mainTemplate = `package main

import (
	"log"

	"{{ .ModuleName }}/internal/controller"
	"{{ .ModuleName }}/internal/route"
	"github.com/klass-lk/ginboot"
)

func main() {
	// Create a new ginboot server
	server := ginboot.NewServer()

	// Initialize and register controllers
	pingController := controller.NewPingController()
	server.RegisterController("/ping", pingController)

	// Setup routes
	route.SetupRoutes(server.Router)

	if err := server.Start(8080); err != nil {
		log.Fatal(err)
	}
}`

const goModTemplate = `module {{ .ModuleName }}

go 1.21

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/klass-lk/ginboot v1.0.11
)`

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

const pingControllerTemplate = `package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type PingController struct{}

func NewPingController() *PingController {
	return &PingController{}
}

func (c *PingController) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}`

const routesTemplate = `package route

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	// Add your custom routes here
}`
