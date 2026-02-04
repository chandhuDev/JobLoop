# JobLoop

**JobLoop** is an intelligent, high-performance web scraping platform that automates the discovery of startup companies and their job openings from job provider portals. Built with concurrency at its core, JobLoop simultaneously scrapes company data, discovers testimonial images using AI vision, and aggregates job listings—all while serving real-time data through a REST API.

## Overview

JobLoop crawls startup directories like **Y Combinator** and **Peerlist** to discover seed companies. For each seed company, it scrapes testimonial images from their websites, uses Anthropic's Claude Vision AI to extract company names mentioned in testimonials, then uses Claude Search to discover URLs for those companies, creating a growing network of discovered companies. Simultaneously, it scrapes job postings from all companies. All data is stored in PostgreSQL and exposed via a clean REST API.

### Key Features

- **Multi-Source Scraping**: Seeds initial company discovery from Y Combinator and Peerlist
- **Recursive Company Discovery**: Extracts new companies from testimonials, creating a self-expanding network
- **Concurrent Processing**: Scrapes jobs and testimonials in parallel using Go's goroutines
- **AI-Powered Vision**: Uses Anthropic's Claude Vision API to analyze testimonial images and extract company names
- **Claude Search Integration**: Leverages Anthropic's Claude Search to find URLs for discovered companies
- **Headless Browser Automation**: Playwright-powered scraping handles JavaScript-rendered content
- **RESTful API**: Query companies, jobs, and statistics through well-defined endpoints
- **PostgreSQL Storage**: Robust relational database with proper indexing and constraints
- **Docker Support**: Containerized deployment with all dependencies included
- **Structured Logging**: JSON-based logging with zerolog for production monitoring

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        JobLoop Scraper                           │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │ Y Combinator │    │  Peerlist    │    │   HTTP API   │      │
│  │   Scraper    │    │   Scraper    │    │   Server     │      │
│  └──────┬───────┘    └──────┬───────┘    └──────────────┘      │
│         │                   │                                   │
│         └───────┬───────────┘                                   │
│                 ▼                                                │
│     ┌───────────────────────┐                                   │
│     │ Playwright Browser    │                                   │
│     │    (Chromium)         │                                   │
│     └───────────────────────┘                                   │
│                 │                                                │
│                 ▼                                                │
│     ┌───────────────────────┐                                   │
│     │  SEED COMPANIES (DB)  │◄────────────────┐                │
│     │  (Root Companies)     │                 │                │
│     └───────────┬───────────┘                 │                │
│                 │                              │                │
│         ┌───────┴────────┐                     │                │
│         ▼                ▼                     │                │
│  ┌─────────────┐  ┌──────────────┐            │                │
│  │     Job     │  │ Testimonial  │            │                │
│  │   Scraper   │  │   Scraper    │            │                │
│  └──────┬──────┘  └──────┬───────┘            │                │
│         │                │                     │                │
│         │                ▼                     │                │
│         │         ┌──────────────┐             │                │
│         │         │   Anthropic  │             │                │
│         │         │ Vision API   │             │                │
│         │         │ (Extract Co.)│             │                │
│         │         └──────┬───────┘             │                │
│         │                │                     │                │
│         │                ▼                     │                │
│         │         ┌──────────────┐             │                │
│         │         │   Anthropic  │             │                │
│         │         │ Claude Search│             │                │
│         │         │ (Find URLs)  │             │                │
│         │         └──────┬───────┘             │                │
│         │                │                     │                │
│         │                └─────────────────────┘                │
│         │                     (New Seed Companies)              │
│         ▼                                                       │
│  ┌─────────────────────────┐                                   │
│  │   PostgreSQL Database   │                                   │
│  │  - seed_companies       │                                   │
│  │  - jobs                 │                                   │
│  │  - testimonial_companies│                                   │
│  └─────────────────────────┘                                   │
└──────────────────────────────────────────────────────────────────┘
```

## Technology Stack

- **Language**: Go 1.25
- **Database**: PostgreSQL with GORM ORM
- **Browser Automation**: Playwright (Chromium)
- **AI/ML**:
  - Anthropic Claude Vision API (testimonial analysis)
  - Anthropic Claude Search (company URL discovery)
- **Logging**: zerolog with file rotation (lumberjack)
- **Containerization**: Docker with multi-stage builds
- **Concurrency**: Native Go goroutines, sync primitives, and errgroups

## Prerequisites

Before setting up JobLoop locally, ensure you have the following installed:

- **Go 1.25+** - [Download](https://golang.org/dl/)
- **PostgreSQL 14+** - [Download](https://www.postgresql.org/download/)
- **Docker & Docker Compose** (optional, for containerized setup) - [Download](https://www.docker.com/get-started)
- **Git** - [Download](https://git-scm.com/downloads)

### API Keys Required

You'll need to obtain the following API key:

**Anthropic API Key** - For Claude Vision AI and Claude Search
- Sign up at [Anthropic Console](https://console.anthropic.com/)
- Create a new API key
- This single key is used for both Vision API (testimonial analysis) and Search API (URL discovery)

## Installation & Setup

### 1. Clone the Repository

```bash
git clone https://github.com/chandhuDev/JobLoop.git
cd JobLoop
```

### 2. Set Up PostgreSQL Database

#### Option A: Local PostgreSQL

```bash
# Create the database
psql -U postgres
CREATE DATABASE jobloop;
\q
```

#### Option B: Using Docker

```bash
docker run -d \
  --name jobloop-postgres \
  -e POSTGRES_PASSWORD=yourpassword \
  -e POSTGRES_DB=jobloop \
  -p 5432:5432 \
  postgres:15-alpine
```

### 3. Configure Environment Variables

Create a `.env` file in the project root:

```bash
cp .env.example .env  # If example exists, otherwise create manually
```

Edit `.env` with your configuration:

```env
# Required API Key
ANTHROPIC_API_KEY=your_anthropic_api_key_here

# Database Configuration
DB_LOCAL_HOST=localhost
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=jobloop

# Optional: For Docker deployments
DB_HOST=jobloop-postgres
DB_PORT=5432
```

### 4. Install Go Dependencies

```bash
go mod download
```

### 5. Install Playwright Browsers

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps chromium
```

This downloads the Chromium browser and required system dependencies.

### 6. Run the Application

```bash
# Build the application
go build -o bin/jobloop ./cmd/api/

# Run it
./bin/jobloop
```

Or run directly:

```bash
go run ./cmd/api/main.go
```

You should see output like:

```
{"level":"info","time":"2026-02-03T...","message":"seed company scraper started"}
{"level":"info","time":"2026-02-03T...","message":"Starting HTTP server","addr":":8081"}
{"level":"info","time":"2026-02-03T...","message":"worker started for ycombinator"}
```

### 7. Verify Installation

Test the API:

```bash
# Health check
curl http://localhost:8081/health

# Get statistics
curl http://localhost:8081/api/state

# List companies (after scraping completes)
curl http://localhost:8081/api/companies?limit=10

# List jobs
curl http://localhost:8081/api/jobs?limit=10
```

## Docker Deployment

### Build and Run with Docker

```bash
# Build the Docker image
docker build -t jobloop:latest .

# Run the container
docker run -d \
  --name jobloop \
  -p 8081:8081 \
  -e ANTHROPIC_API_KEY=your_key \
  -e DB_HOST=your_postgres_host \
  -e DB_USER=postgres \
  -e DB_PASSWORD=yourpassword \
  -e DB_NAME=jobloop \
  jobloop:latest
```

### Using Docker Compose (Recommended)

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: jobloop-postgres
    environment:
      POSTGRES_DB: jobloop
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: yourpassword
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  jobloop:
    build: .
    container_name: jobloop-app
    ports:
      - "8081:8081"
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: yourpassword
      DB_NAME: jobloop
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

volumes:
  postgres_data:
```

Run with:

```bash
docker-compose up -d
```

## API Reference

The HTTP server runs on port **8081** and provides the following endpoints:

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "time": "2026-02-03T10:30:00Z"
}
```

### Get Database Statistics

```http
GET /api/state
```

**Response:**
```json
{
  "companies": 50,
  "jobs": 1247,
  "timestamp": "2026-02-03T10:30:00Z"
}
```

### List Companies

```http
GET /api/companies?limit=50&offset=0
```

**Query Parameters:**
- `limit` (optional): Number of results (1-100, default: 50)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "company_name": "Acme Corp",
      "company_url": "https://acme.com",
      "visited": true,
      "testimonial_scraped": true,
      "job_scraped": true,
      "created_at": "2026-02-03T10:00:00Z"
    }
  ],
  "total": 50,
  "limit": 50,
  "offset": 0
}
```

### List Jobs

```http
GET /api/jobs?limit=50&offset=0&company_id=1
```

**Query Parameters:**
- `limit` (optional): Number of results (1-100, default: 50)
- `offset` (optional): Pagination offset (default: 0)
- `company_id` (optional): Filter by specific company

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "seed_company_id": 1,
      "job_title": "Senior Software Engineer",
      "job_url": "https://acme.com/careers/senior-swe",
      "created_at": "2026-02-03T10:15:00Z"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

## Project Structure

```
JobLoop/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration files
│   ├── database/
│   │   └── database_service.go  # Database connection & setup
│   ├── interfaces/              # Interface definitions
│   ├── logger/
│   │   └── logger.go           # Structured logging setup
│   ├── models/                  # Data models & DTOs
│   ├── repository/              # Database operations
│   │   ├── job_repo.go
│   │   ├── seed_company_repo.go
│   │   └── testimonial_repo.go
│   ├── schema/
│   │   └── schema.go           # GORM database schemas
│   └── service/
│       ├── browser_service.go   # Playwright browser management
│       ├── error_service.go     # Error handling
│       ├── http_handler_service.go  # API endpoints
│       ├── scraper_service.go   # Job scraping logic
│       ├── search_service.go    # Claude Search integration
│       ├── seed_company_service.go  # Company scraping
│       ├── testimonial_service.go   # Testimonial scraping
│       └── vision_service.go    # Claude Vision AI integration
├── logs/                        # Application logs (gitignored)
├── .env                        # Environment variables (gitignored)
├── .gitignore
├── Dockerfile                   # Docker build configuration
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── README.md                   # This file
```

## How It Works

### 1. Seed Company Discovery (Initial Bootstrap)

JobLoop starts by scraping **seed companies** from:
- **Y Combinator Companies Directory** (`/companies`)
- **Peerlist Jobs Board** (`/jobs`)

For each source, it:
- Uses Playwright to navigate to the listing page
- Waits for JavaScript-rendered content to load
- Extracts company names and URLs
- Stores companies in PostgreSQL as **seed companies** with unique constraints

### 2. Job Scraping (Concurrent)

For each seed company, JobLoop concurrently:
- Searches for the company's careers page
- Scrapes available job listings (title, URL)
- Stores jobs with a composite unique index on `(seed_company_id, job_title)`
- Handles missing careers pages gracefully

### 3. Recursive Company Discovery via Testimonials (The Growth Engine)

In parallel, the testimonial scraper creates a **self-expanding company network**:

1. **Scrape Testimonials**: For each seed company, scrapes testimonial images from their website
2. **Extract Companies**: Uses Claude Vision API to analyze testimonial images and extract company names mentioned
3. **Find URLs**: Uses Claude Search to discover URLs for the extracted company names
4. **Create New Seeds**: Stores discovered companies as **new seed companies** in the database
5. **Repeat**: These new seed companies feed back into the job scraping and testimonial discovery cycle

This creates a recursive discovery loop where companies lead to more companies.

### 4. Concurrency Model

- **Goroutines**: Each company scraper runs in its own goroutine
- **Wait Groups**: Coordinates completion of scraping batches
- **Channels**: Passes seed company data between scraper stages
- **Error Groups**: Manages HTTP server and scraper lifecycles
- **Atomic Counters**: Limits max companies processed per batch (configurable)

### 5. Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Initial Seed Companies                   │
│              (Y Combinator, Peerlist, etc.)                 │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
         ┌───────────────────────────────┐
         │  Store as Seed Companies (DB) │
         └───────────┬───────────────────┘
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
┌──────────────────┐    ┌──────────────────┐
│   Job Scraper    │    │ Testimonial      │
│                  │    │ Scraper          │
│ - Find careers   │    │                  │
│ - Extract jobs   │    │ 1. Scrape images │
│ - Store in DB    │    │ 2. Vision API    │
└──────────────────┘    │    (extract co.) │
                        │ 3. Claude Search │
                        │    (find URLs)   │
                        └────────┬─────────┘
                                 │
                                 ▼
                        ┌──────────────────┐
                        │ New Companies    │
                        │ (Back to DB as   │
                        │  Seed Companies) │
                        └────────┬─────────┘
                                 │
                                 └─────► Cycle Repeats
```

## Configuration

### Scraper Limits

Edit `internal/service/seed_company_service.go` to adjust:

```go
// Y Combinator scraper
const maxCompanies = 50  // Line 123

// Peerlist scraper (in UploadSeedCompanyToChannel)
const maxCompanies = 15  // Line 236
```

### Scraper Sources

Modify `cmd/api/main.go` (lines 140-153) to add/remove sources:

```go
SeedCompanyConfigs := []models.SeedCompany{
    {
        Name:     "Y Combinator",
        URL:      "http://www.ycombinator.com/companies",
        Selector: `a[href^="/companies/"]`,
        WaitTime: 3 * time.Second,
    },
    // Add more sources here
}
```

### Database Schema

The application auto-migrates three tables:
- `seed_companies` - Root companies and recursively discovered companies
- `jobs` - Job listings scraped from seed companies
- `testimonial_companies` - Companies extracted from testimonials (before becoming seed companies)

## Development

### Running Tests

```bash
go test ./...
```

### Build for Production

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o bin/jobloop ./cmd/api/
```

### Logging

Logs are written to:
- **stdout** (JSON format for production)
- **logs/app.log** (file rotation enabled, max 100MB, 30 days retention)

View live logs:

```bash
tail -f logs/app.log | jq
```

## Contributing

We welcome contributions to JobLoop! Here's how to get started:

### 1. Fork & Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/JobLoop.git
cd JobLoop
git remote add upstream https://github.com/chandhuDev/JobLoop.git
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 3. Set Up Development Environment

Follow the [Installation & Setup](#installation--setup) section above.

### 4. Make Your Changes

- Write clean, idiomatic Go code
- Follow existing code structure and naming conventions
- Add comments for complex logic
- Update tests if applicable

### 5. Test Your Changes

```bash
# Run the application
go run ./cmd/api/main.go

# Verify API endpoints
curl http://localhost:8081/health
curl http://localhost:8081/api/companies
```

### 6. Commit Your Changes

```bash
git add .
git commit -m "feat: add your feature description"
```

Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

### 7. Push & Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then open a Pull Request on GitHub with:
- Clear description of changes
- Any related issue numbers
- Screenshots (if UI changes)

### Development Guidelines

- **Code Style**: Run `gofmt` and `golint` before committing
- **Error Handling**: Always handle errors explicitly, never use `_` unless justified
- **Logging**: Use structured logging with appropriate levels (Info, Warn, Error)
- **Concurrency**: Document any goroutines, channels, or sync primitives
- **Database**: Use GORM best practices, avoid N+1 queries
- **API**: Maintain backward compatibility for existing endpoints

### Useful Make Commands (if Makefile exists)

```bash
make build    # Build binary
make run      # Run application
make test     # Run tests
make docker   # Build Docker image
make clean    # Clean build artifacts
```

## Troubleshooting

### Issue: "Found companies 0" for Y Combinator

**Cause**: Y Combinator's page is JavaScript-rendered and takes time to load.

**Solution**: The selector might have changed. Inspect the page manually:
1. Visit `https://www.ycombinator.com/companies`
2. Right-click on a company → Inspect
3. Find the correct CSS selector
4. Update in `cmd/api/main.go` line 144

### Issue: Playwright browser fails to launch

**Cause**: Missing system dependencies.

**Solution**:
```bash
# Re-install with dependencies
go run github.com/playwright-community/playwright-go/cmd/playwright install --with-deps chromium

# On Linux, you may need:
sudo apt-get install -y libgbm1 libnss3 libatk1.0-0
```

### Issue: Database connection failed

**Cause**: PostgreSQL not running or wrong credentials.

**Solution**:
```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Verify credentials in .env match your PostgreSQL setup
# Try connecting manually:
psql -h localhost -U postgres -d jobloop
```

### Issue: Rate limiting from APIs

**Cause**: Too many concurrent requests to Anthropic APIs.

**Solution**: Reduce `maxCompanies` limits or add delays between requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Playwright Go](https://github.com/playwright-community/playwright-go) - Browser automation
- [Anthropic](https://www.anthropic.com/) - Claude Vision AI and Claude Search
- [GORM](https://gorm.io/) - Go ORM library
- [zerolog](https://github.com/rs/zerolog) - Structured logging

## Support

For issues, questions, or contributions:
- Open an issue on [GitHub Issues](https://github.com/chandhuDev/JobLoop/issues)
- Start a discussion on [GitHub Discussions](https://github.com/chandhuDev/JobLoop/discussions)

---

**Built with ❤️ using Go, PostgreSQL, and AI**
