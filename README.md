# 🏛️ GOV.UK Reports Dashboard

> A reporting dashboard for monitoring GOV.UK services and infrastructure

A Go app providing reporting capabilities for GOV.UK Web, Publishing, and Platform services. The dashboard integrates multiple monitoring systems and provides both web interfaces and REST APIs for infrastructure and application services.

![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)
![Build Status](https://img.shields.io/badge/Build-Passing-green.svg)

## ✨ Features

### 💰 **Cost Reporting**

- **AWS Cost Explorer integration** for real-time cost monitoring
- **Application-level cost breakdown** grouped by GOV.UK services

### 🗄️ **RDS Version Checker**

- **PostgreSQL instance discovery** from AWS RDS
- **Version compliance monitoring** with EOL tracking
- **End-of-life detection** with immediate alerts
- **Detailed instance specifications** and metadata

## 🏗️ Architecture

### **Modular Reports Framework**

```
📊 Reports Dashboard
├── 💰 Cost Reporter (AWS Cost Explorer)
├── 🗄️ RDS Version Checker (PostgreSQL monitoring)
└── 🔌 Extensible framework for new modules
```

### **Directory Structure**

```
├── cmd/server/              # Application entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── handlers/           # Core HTTP handlers and middleware
│   ├── models/             # Shared data structures
│   ├── modules/            # Report modules
│   │   ├── costs/          # Cost reporting module
│   │   └── rds/            # RDS monitoring module
│   └── reports/            # Reports framework
│       ├── types.go        # Report interfaces
│       ├── manager.go      # Module registry
│       ├── renderer.go     # Common utilities
│       └── cache.go        # Caching system
├── pkg/
│   ├── aws/               # AWS client integration
│   ├── govuk/             # GOV.UK API client
│   └── common/            # Shared types
└── web/
    ├── static/            # CSS/JS assets
    └── templates/         # HTML templates
```

## 🚀 Quick Start

### **Prerequisites**

- Go 1.21 or later
- AWS credentials configured
- Access to AWS Cost Explorer API
- Access to AWS RDS (optional)

### **1. Setup Environment**

```bash
# Clone the repository
git clone git://github.com/alphagov/govuk-reports-prototype.git
cd govuk-reports-prototype

# Install dependencies
go mod tidy

# Set up development environment
make setup
```

### **2. Configure AWS Credentials**

```bash
# Option 1: AWS Profile (Recommended)
export AWS_PROFILE=your-profile-name

# Option 2: Direct credentials
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_REGION=eu-west-2
```

### **3. Start the Application**

```bash
# Using Make (recommended)
make run

# Or manually
go run cmd/server/main.go
```

### **4. Access the Dashboard**

Open your browser to `http://localhost:8080`

## 🌐 API Reference

### **Core Endpoints**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | 🎨 Main dashboard with all report modules |
| `/api/health` | GET | 🏥 Service health check |

### **Cost Reporting APIs**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/applications` | GET | 📋 List all applications with costs |
| `/api/applications/{name}` | GET | 🔍 Get specific application details |
| `/api/applications/{name}/services` | GET | ⚙️ Get application service breakdown |
| `/api/costs` | GET | 💰 Legacy cost summary (backwards compatibility) |
| `/api/costs/summary` | GET | 💰 Cost module summary |

### **RDS Monitoring APIs**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/rds/health` | GET | 🏥 RDS service health check |
| `/api/rds/summary` | GET | 📊 RDS summary statistics |
| `/api/rds/instances` | GET | 🗄️ List PostgreSQL instances |
| `/api/rds/instances/{id}` | GET | 🔍 Get specific instance details |
| `/api/rds/versions` | GET | 📋 Version check results |
| `/api/rds/outdated` | GET | ⚠️ Outdated/EOL instances |

### **Reports Framework APIs**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/reports/list` | GET | 📋 List available reports with metadata |
| `/api/reports/summary` | GET | 📊 Dashboard summary for all reports |
| `/api/reports/{id}` | GET | 🔍 Get specific report by ID |
| `/api/reports/costs` | GET | 💰 Cost report via framework |
| `/api/reports/rds` | GET | 🗄️ RDS report via framework |

## 🎯 Usage Examples

### **Cost Reporting**

```bash
# Get all applications with costs
curl http://localhost:8080/api/applications

# Get specific application
curl http://localhost:8080/api/applications/publishing-api

# Get cost summary
curl http://localhost:8080/api/costs/summary
```

**Example Response:**

```json
{
  "applications": [
    {
      "name": "Publishing API",
      "shortname": "publishing-api",
      "team": "Publishing Platform",
      "production_hosted_on": "EKS",
      "total_cost": 1250.75,
      "currency": "GBP",
      "service_count": 12
    }
  ],
  "total_cost": 15750.50,
  "count": 24,
  "currency": "GBP"
}
```

### **RDS Version Checking**

```bash
# Get RDS summary
curl http://localhost:8080/api/rds/summary

# Get all PostgreSQL instances
curl http://localhost:8080/api/rds/instances

# Check version compliance
curl http://localhost:8080/api/rds/versions
```

**Example Response:**

```json
{
  "postgresql_count": 15,
  "eol_instances": 2,
  "outdated_instances": 3,
  "version_summary": [
    {
      "major_version": "14",
      "count": 8,
      "is_eol": false,
      "is_outdated": false
    },
    {
      "major_version": "11",
      "count": 2,
      "is_eol": true,
      "is_outdated": false
    }
  ]
}
```

### **Reports Framework**

```bash
# Get all available reports
curl http://localhost:8080/api/reports/list

# Get unified dashboard summary
curl http://localhost:8080/api/reports/summary

# Get specific report
curl http://localhost:8080/api/reports/costs
```

## 🔧 Adding New Report Modules

The Reports Dashboard uses a modular architecture that makes it easy to add new report types.

### **1. Create Module Structure**

```bash
mkdir -p internal/modules/yourmodule
```

### **2. Implement Report Interface**

```go
// internal/modules/yourmodule/report.go
package yourmodule

import (
    "context"
    "time"
    "govuk-reports-dashboard/internal/reports"
)

type YourModuleReport struct {
    service *YourModuleService
    logger  *logger.Logger
}

func (r *YourModuleReport) GetMetadata() reports.ReportMetadata {
    return reports.ReportMetadata{
        ID:          "yourmodule",
        Name:        "Your Module Name",
        Description: "Description of what this module does",
        Type:        reports.ReportTypeHealth,
        Version:     "1.0.0",
        Priority:    reports.PriorityMedium,
    }
}

func (r *YourModuleReport) GenerateSummary(ctx context.Context, params reports.ReportParams) ([]reports.Summary, error) {
    // Implement summary generation
}

func (r *YourModuleReport) GenerateReport(ctx context.Context, params reports.ReportParams) (reports.ReportData, error) {
    // Implement detailed report generation
}

func (r *YourModuleReport) IsAvailable(ctx context.Context) bool {
    // Check if the module can run
}

func (r *YourModuleReport) GetRefreshInterval() time.Duration {
    return 15 * time.Minute
}

func (r *YourModuleReport) Validate(params reports.ReportParams) error {
    // Validate parameters
    return nil
}
```

### **3. Register Module**

```go
// In cmd/server/main.go
yourModuleService := yourmodule.NewService(dependencies...)
yourModuleReport := yourmodule.NewReport(yourModuleService, log)

err = reportsManager.Register(yourModuleReport)
if err != nil {
    log.WithError(err).Error().Msg("Failed to register your module")
}
```

### **4. Add Routes**

```go
// Add API routes
yourModule := api.Group("/yourmodule")
{
    yourModule.GET("/health", yourModuleHandler.GetHealth)
    yourModule.GET("/summary", yourModuleHandler.GetSummary)
}

// Add web routes
router.GET("/yourmodule", yourModuleHandler.GetPage)
```

## 🛠️ Development

### **Quality Assurance**

```bash
# Run all quality checks
make check

# Run tests with coverage
make test-coverage

# Security scanning
make security

# Code formatting
make fmt
```

### **Building & Running**

```bash
# Development mode
make run

# Build for production
make build

# Docker deployment
make docker-build
make docker-run
```

### **Environment Configuration**

```bash
# Copy example configuration
make env-example
cp .env.example .env

# Edit configuration
vim .env
```

## 📊 Configuration

### **Server Configuration**

- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - Environment mode (default: development)
- `READ_TIMEOUT` - HTTP read timeout (default: 30s)
- `WRITE_TIMEOUT` - HTTP write timeout (default: 30s)

### **AWS Configuration**

- `AWS_REGION` - AWS region (default: eu-west-2)
- `AWS_PROFILE` - AWS profile for credentials
- `AWS_ACCESS_KEY_ID` - Direct AWS access key
- `AWS_SECRET_ACCESS_KEY` - Direct AWS secret key

### **Reports Configuration**

- `REPORTS_CACHE_TTL` - Cache time-to-live (default: 15m)
- `REPORTS_MAX_CONCURRENT` - Max concurrent reports (default: 10)

### **Logging Configuration**

- `LOG_LEVEL` - Log level (debug, info, warn, error)
- `LOG_FORMAT` - Log format (json, text)

## 🔍 Monitoring & Health Checks

### **Service Health**

```bash
# Overall system health
curl http://localhost:8080/api/health

# RDS service health
curl http://localhost:8080/api/rds/health

# Reports framework status
curl http://localhost:8080/api/reports/list
```

## 🤝 Contributing

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/new-module`
3. **Implement** following the modular architecture
4. **Test** thoroughly: `make test`
5. **Quality** check: `make check-all`
6. **Commit** with conventional commits
7. **Submit** Pull Request

## 📜 License

**Crown Copyright (C) 2025**

This project is licensed under Crown Copyright. See the LICENSE file for details.

---
