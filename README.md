# GOV.UK AWS Cost Dashboard

A Golang web application for monitoring and displaying AWS costs for GOV.UK services.

## Features

- **AWS Cost Tracking**: Integration with AWS Cost Explorer API
- **GOV.UK API Integration**: Placeholder for GOV.UK specific data
- **Web Dashboard**: Clean, accessible interface following GOV.UK Design System
- **Health Monitoring**: Built-in health check endpoints
- **Structured Logging**: JSON logging with configurable levels
- **Graceful Shutdown**: Proper signal handling and resource cleanup

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
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `AWS_SESSION_TOKEN` - AWS session token (optional)

### GOV.UK Configuration
- `GOVUK_API_BASE_URL` - GOV.UK API base URL
- `GOVUK_API_KEY` - GOV.UK API key

### Logging Configuration
- `LOG_LEVEL` - Log level (default: info)
- `LOG_FORMAT` - Log format: json or text (default: json)

## Running the Application

### Prerequisites
- Go 1.21 or later
- AWS credentials configured
- Docker (optional)

### Local Development

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Set environment variables (create a `.env` file or export them)
4. Run the application:
   ```bash
   go run cmd/server/main.go
   ```

The server will start on port 8080 by default.

### Using Docker

1. Build the Docker image:
   ```bash
   docker build -t govuk-cost-dashboard .
   ```

2. Run the container:
   ```bash
   docker run -p 8080:8080 \
     -e AWS_REGION=eu-west-2 \
     -e AWS_ACCESS_KEY_ID=your_key \
     -e AWS_SECRET_ACCESS_KEY=your_secret \
     govuk-cost-dashboard
   ```

## API Endpoints

- `GET /api/v1/health` - Health check endpoint
- `GET /api/v1/costs` - Get cost summary data
- `GET /` - Web dashboard interface

## Development

### Project Structure

The project follows Go best practices with clear separation of concerns:

- **cmd/**: Application entrypoints
- **internal/**: Private application code
- **pkg/**: Public library code that could be imported by other projects
- **web/**: Web assets and templates

### Adding New Features

1. Add models in `internal/models/`
2. Implement business logic in `internal/services/`
3. Create HTTP handlers in `internal/handlers/`
4. Update routing in `cmd/server/main.go`

### Testing

```bash
go test ./...
```

### Building

```bash
go build -o govuk-cost-dashboard cmd/server/main.go
```

## Security Considerations

- The application runs as a non-root user in Docker
- CORS middleware is configured
- Error handling middleware prevents information leakage
- Structured logging for audit trails

## License

Crown Copyright (C) 2024