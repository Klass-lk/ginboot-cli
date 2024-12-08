# Ginboot CLI

Ginboot CLI is a powerful command-line tool for creating and managing Ginboot framework projects. It provides a streamlined way to scaffold new projects, build applications, and deploy to AWS Lambda.

## Installation

```bash
go install github.com/klass-lk/ginboot-cli@latest
```

## Prerequisites

- Go 1.21 or later
- AWS SAM CLI (for deployment)
- AWS credentials configured

## Usage

### Creating a New Project

Create a new Ginboot project with a standard directory structure:

```bash
ginboot new myproject
```

This will create a new project with the following structure:
```
myproject/
├── internal/
│   ├── controller/
│   │   └── user_controller.go
│   ├── model/
│   │   └── user.go
│   ├── repository/
│   │   └── user_repository.go
│   └── service/
│       └── user_service.go
├── main.go
├── go.mod
├── Makefile
└── template.yaml
```

### Building the Project

Build your project using AWS SAM:

```bash
cd myproject
ginboot build
```

The build process will:
1. Compile your Go application
2. Create a deployment package
3. Store build artifacts in `.aws-sam/build/`

## Deployment Options

### Docker Deployment

The project includes Docker and docker-compose configurations for easy deployment:

1. Build and run with docker-compose:
```bash
docker-compose up --build
```

This will:
- Build the Go application
- Start a MongoDB container
- Create a Docker network
- Set up volume for MongoDB data persistence
- Expose the API on port 8080

2. Access your API at `http://localhost:8080/api/v1`

To run in detached mode:
```bash
docker-compose up -d
```

To stop the services:
```bash
docker-compose down
```

### AWS Lambda Deployment

Deploy your application to AWS Lambda:

```bash
ginboot deploy
```

On first run, you'll be prompted for:
- Stack name (defaults to project name)
- AWS Region (defaults to us-east-1)
- S3 bucket configuration (can use SAM's default bucket)

These settings will be saved in `ginboot-app.yml` for future deployments.

## Project Structure

### Controllers
Controllers handle HTTP requests and define API endpoints. Example:
```go
func (c *UserController) Register(group *ginboot.ControllerGroup) {
    group.GET("/:id", c.GetUser)
    group.POST("", c.CreateUser)
}
```

### Models
Models define your data structures and MongoDB document mappings:
```go
type User struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}
```

### Repositories
Repositories handle database operations using Ginboot's MongoDB integration:
```go
type UserRepository struct {
    *ginboot.MongoRepository[model.User]
}
```

### Services
Services contain your business logic:
```go
type UserService interface {
    GetUser(id string) (model.User, error)
    CreateUser(user model.User) error
}
```

## Configuration

### ginboot-app.yml
Deployment configuration is stored in `ginboot-app.yml`:
```yaml
stack_name: myproject
region: us-east-1
use_default_bucket: true
```

### template.yaml
AWS SAM template defining your Lambda function and API Gateway:
```yaml
Resources:
  MyProjectFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: bootstrap
      Runtime: provided.al2
      Events:
        ApiEvents:
          Type: Api
          Properties:
            Path: /{proxy+}
            Method: ANY
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
