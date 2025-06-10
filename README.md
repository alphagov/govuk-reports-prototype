# 🏛️ GOV.UK AWS Cost Dashboard

A powerful Golang web application for monitoring and displaying AWS costs for GOV.UK services with beautiful dashboards and comprehensive API integration.

![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)
![License](https://img.shields.io/badge/License-Crown%20Copyright-gold.svg)
![Build Status](https://img.shields.io/badge/Build-Passing-green.svg)

## ✨ Features

- 💰 **AWS Cost Tracking**: Real-time integration with AWS Cost Explorer API
- 🏛️ **GOV.UK Apps Integration**: Comprehensive GOV.UK applications API client
- 🎨 **Beautiful Dashboard**: Clean, accessible interface following GOV.UK Design System
- 🔒 **MFA Support**: Full AWS Multi-Factor Authentication support
- 📊 **Health Monitoring**: Built-in health check endpoints with detailed status
- 📝 **Structured Logging**: JSON logging with configurable levels and debugging
- 🛡️ **Graceful Shutdown**: Proper signal handling and resource cleanup
- 🐳 **Docker Ready**: Multi-stage Docker builds with security best practices
- ⚡ **Caching**: Intelligent in-memory caching with configurable TTL
- 🔄 **Retry Logic**: Robust HTTP clients with exponential backoff

## Architecture

```
├── cmd/server/          # Application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── handlers/       # HTTP handlers and middleware
│   ├── models/         # Data structures
│   └── services/       # Business logic
├── pkg/
│   ├── aws/           # AWS client
│   └── govuk/         # GOV.UK API client
└── web/
    ├── static/        # CSS/JS assets
    └── templates/     # HTML templates
```

## Configuration

The application uses environment variables for configuration:

### Server Configuration
- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - Environment mode (default: development)
- `READ_TIMEOUT` - HTTP read timeout in seconds (default: 30)
- `WRITE_TIMEOUT` - HTTP write timeout in seconds (default: 30)

### AWS Configuration
- `AWS_REGION` - AWS region (default: eu-west-2)
- `AWS_PROFILE` - AWS profile name to use from ~/.aws/credentials
- `AWS_ACCESS_KEY_ID` - AWS access key (alternative to profile)
- `AWS_SECRET_ACCESS_KEY` - AWS secret key (alternative to profile)
- `AWS_SESSION_TOKEN` - AWS session token (optional)
- `AWS_MFA_TOKEN` - MFA token for assume role operations (optional)

### GOV.UK Configuration
- `GOVUK_API_BASE_URL` - GOV.UK API base URL
- `GOVUK_API_KEY` - GOV.UK API key

### Logging Configuration
- `LOG_LEVEL` - Log level (default: info)
- `LOG_FORMAT` - Log format: json or text (default: json)

## Running the Application

### Prerequisites
- Go 1.21 or later
- AWS credentials configured (see AWS Configuration section)
- Docker (optional)

### AWS Credential Setup

The application supports multiple ways to configure AWS credentials:

#### Option 1: AWS Profile (Recommended)
```bash
# Set the profile name
export AWS_PROFILE=your-profile-name

# Or create/update ~/.aws/credentials file:
[your-profile-name]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
region = eu-west-2
```

#### Option 2: Environment Variables
```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=eu-west-2
```

#### Option 3: EC2 Instance Role (for production)
When running on EC2, the application will automatically use the instance role.

#### MFA Support
If your AWS profile requires MFA, the application supports it in two ways:

**Interactive MFA (for development):**
The application will prompt you for the MFA token when needed.

**Non-interactive MFA (for automation):**
```bash
export AWS_MFA_TOKEN=123456
AWS_PROFILE=your-profile-name go run cmd/server/main.go
```

### 🚀 Quick Start

The easiest way to get started is with our Makefile:

```bash
# Set up your development environment
make setup

# Create environment configuration
make env-example
cp .env.example .env
# Edit .env with your settings

# Run the application
AWS_PROFILE=your-profile make run
```

### 🛠️ Development Commands

We provide a comprehensive Makefile with all common tasks:

```bash
# 🚀 Running
make run                    # Run the application
make run-example           # Run GOV.UK apps example
make run-with-profile      # Run with specific AWS profile

# 🧪 Testing  
make test                  # Run all tests
make test-coverage         # Generate coverage report
make test-race             # Test with race detection
make test-govuk           # Test GOV.UK client only

# 🔧 Development
make build                 # Build the application
make fmt                   # Format code
make vet                   # Run go vet
make lint                  # Run linter
make check                 # Run all quality checks

# 🐳 Docker
make docker-build          # Build Docker image
make docker-run            # Run in container

# 📖 Help
make help                  # Show all available commands
```

### Manual Development Setup

If you prefer manual setup:

1. Clone the repository
2. Install dependencies: `go mod tidy`
3. Configure AWS credentials (see AWS Credential Setup above)
4. Set environment variables (create a `.env` file or export them)
5. Run: `go run cmd/server/main.go`

The server will start on port 8080 by default.

### 🐳 Using Docker

Using the Makefile (recommended):
```bash
make docker-build
make docker-run
```

Or manually:
```bash
docker build -t govuk-cost-dashboard .
docker run -p 8080:8080 \
  -e AWS_REGION=eu-west-2 \
  -e AWS_PROFILE=your-profile \
  govuk-cost-dashboard
```

## 🌐 API Endpoints

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/` | GET | 🎨 Web dashboard interface | `curl http://localhost:8080/` |
| `/api/v1/health` | GET | 🏥 Health check endpoint | `curl http://localhost:8080/api/v1/health` |
| `/api/v1/costs` | GET | 💰 Cost summary data | `curl http://localhost:8080/api/v1/costs` |
| `/static/*` | GET | 📁 Static assets (CSS/JS) | `curl http://localhost:8080/static/css/styles.css` |

## 🛠️ Development

### 📁 Project Structure

The project follows Go best practices with clear separation of concerns:

```
govuk-cost-dashboard/
├── 📁 cmd/server/          # 🚀 Application entry point
├── 📁 internal/            # 🔒 Private application code
│   ├── config/            # ⚙️  Configuration management  
│   ├── handlers/          # 🌐 HTTP handlers & middleware
│   ├── models/            # 📊 Data structures
│   └── services/          # 🔧 Business logic
├── 📁 pkg/                # 📦 Public library code
│   ├── aws/               # ☁️  AWS client
│   └── govuk/             # 🏛️  GOV.UK API client
├── 📁 web/                # 🎨 Web assets
│   ├── static/            # 📄 CSS/JS files
│   └── templates/         # 📝 HTML templates
├── 📁 examples/           # 📚 Usage examples
└── 🐳 Dockerfile          # Docker configuration
```

### 🔧 Adding New Features

1. **Models**: Add data structures in `internal/models/`
2. **Services**: Implement business logic in `internal/services/`
3. **Handlers**: Create HTTP handlers in `internal/handlers/`
4. **Routes**: Update routing in `cmd/server/main.go`
5. **Tests**: Add tests alongside your code

### 🧪 Testing

```bash
# Run all tests
make test

# Generate coverage report  
make test-coverage

# Test with race detection
make test-race

# Quick tests only
make test-short
```

### 🔨 Building

```bash
# Build main application
make build

# Build all binaries (including examples)
make build-all

# Clean build artifacts
make clean
```

## 🔒 Security Considerations

- 🛡️ **Container Security**: Application runs as non-root user in Docker
- 🌐 **CORS Protection**: Configurable CORS middleware
- 🕵️ **Information Leakage**: Error handling middleware prevents sensitive data exposure
- 📋 **Audit Trails**: Comprehensive structured logging for security monitoring
- 🔐 **MFA Support**: Full AWS Multi-Factor Authentication integration
- 🔑 **Credential Management**: Multiple secure credential provider options

## 📊 Quality Assurance

Run comprehensive quality checks:

```bash
# Quick quality check
make check

# Full quality audit
make check-all

# Pre-commit checks
make pre-commit

# Security scanning (requires gosec)
make security
```

## 📈 Project Statistics

Want to see some fun stats about the project?

```bash
make stats
```

## 🤝 Contributing

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Quality**: Run `make check-all` to ensure code quality
4. **Commit**: Use conventional commits: `git commit -m 'feat: add amazing feature'`
5. **Push**: `git push origin feature/amazing-feature`
6. **PR**: Open a Pull Request with detailed description

## 🆘 Troubleshooting

### Common Issues

**MFA Token Issues**
```bash
# Interactive MFA
AWS_PROFILE=your-profile make run

# Non-interactive MFA
AWS_MFA_TOKEN=123456 AWS_PROFILE=your-profile make run
```

**Build Issues**
```bash
# Clean and rebuild
make clean
make deps
make build
```

**Test Failures**
```bash
# Run specific test package
make test-govuk
make test-aws

# Verbose test output
go test -v ./pkg/govuk
```

### Getting Help

- 📖 **Commands**: Run `make help` for all available commands
- 🐹 **Fun**: Run `make gopher` for motivation
- 📊 **Stats**: Run `make stats` for project information
- 📚 **Docs**: Run `make docs` to generate documentation

## 📜 License

**Crown Copyright (C) 2024**

This project is licensed under the Crown Copyright. See the LICENSE file for details.

---

<div align="center">

**🏛️ Built with ❤️ for GOV.UK**

Made with Go • Powered by AWS • Designed for Excellence

</div>