# Docker Setup Guide

JobLoop is now split into two independent services:
- **Server**: Lightweight HTTP API server (port 5001)
- **Scraper**: Browser-based job scraper (runs once and exits)

## Prerequisites

- Docker and Docker Compose installed
- `.env` file with required variables (copy from `.env.example`)

## Quick Start

### 1. Setup Environment

```bash
# Copy and configure environment variables
cp .env.example .env
# Edit .env and add your API keys
```

Required variables:
- `POSTGRES_USER` - Database user
- `POSTGRES_PASSWORD` - Database password
- `ANTHROPIC_API_KEY` - For scraper
- `MAX_LEN` - Maximum companies to scrape

### 2. Start the API Server (with infrastructure)

```bash
# Start server + database + monitoring
docker-compose up -d

# View logs
docker-compose logs -f server
```

This starts:
- ✅ PostgreSQL database
- ✅ HTTP API server (port 5001)
- ✅ PgAdmin (port 8080)
- ✅ Grafana (port 3000)
- ✅ Loki + Promtail (logging)

### 3. Run the Scraper (one-time)

```bash
# Run scraper once
docker-compose --profile scraper up scraper

# Or run in detached mode
docker-compose --profile scraper up -d scraper

# View scraper logs
docker-compose logs -f scraper
```

The scraper will:
- Connect to the database
- Scrape jobs from configured sources
- Insert data into PostgreSQL
- Exit when complete

## Service Management

### Start/Stop/Restart Server

```bash
# Start server only
docker-compose up -d server

# Stop server
docker-compose stop server

# Restart server
docker-compose restart server

# View server logs
docker-compose logs -f server
```

### Run Scraper Periodically

```bash
# Run scraper manually whenever needed
docker-compose --profile scraper up scraper

# Or use cron (example: run daily at 2 AM)
0 2 * * * cd /path/to/JobLoop && docker-compose --profile scraper up scraper >> /var/log/jobloop-scraper.log 2>&1
```

### View All Services

```bash
# List running services
docker-compose ps

# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f server
docker-compose logs -f scraper
docker-compose logs -f postgres
```

## API Endpoints

Once the server is running, access these endpoints:

- `http://localhost:5001/health` - Health check
- `http://localhost:5001/api/companies?limit=50&offset=0` - Get companies (paginated)
- `http://localhost:5001/api/jobs?limit=50&offset=0` - Get jobs (paginated)
- `http://localhost:5001/api/jobs?company_id=123` - Get jobs for specific company
- `http://localhost:5001/api/state` - Database statistics

## Monitoring & Admin Tools

- **PgAdmin**: http://localhost:8080
  - Email: From `PGADMIN_DEFAULT_EMAIL` in `.env`
  - Password: From `PGADMIN_DEFAULT_PASSWORD` in `.env`

- **Grafana**: http://localhost:3000
  - Username: From `GRAFANA_USER` in `.env` (default: admin)
  - Password: From `GRAFANA_PASSWORD` in `.env` (default: admin)

## Building from Source

```bash
# Build both server and scraper
docker-compose build

# Build specific service
docker-compose build server
docker-compose build scraper

# Build without cache
docker-compose build --no-cache
```

## Development

### Run Locally (without Docker)

**Server:**
```bash
# Set environment variables
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=yourpassword

# Run server
go run cmd/server/main.go
```

**Scraper:**
```bash
# Set environment variables
export ANTHROPIC_API_KEY=your_key
export MAX_LEN=200
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=yourpassword

# Run scraper
go run cmd/scraper/main.go
```

## Troubleshooting

### Server won't start
```bash
# Check logs
docker-compose logs server

# Check database is healthy
docker-compose ps postgres

# Restart everything
docker-compose down
docker-compose up -d
```

### Scraper fails
```bash
# Check logs
docker-compose logs scraper

# Common issues:
# - Missing ANTHROPIC_API_KEY
# - Missing MAX_LEN variable
# - Database not accessible
```

### Database issues
```bash
# Connect to database
docker-compose exec postgres psql -U postgres -d jobloop

# Reset database (WARNING: deletes all data)
docker-compose down -v
docker-compose up -d postgres
```

## Cleanup

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (deletes data)
docker-compose down -v

# Remove images
docker-compose down --rmi all
```

## Architecture

```
┌─────────────┐      ┌──────────────┐
│   Client    │─────▶│    Server    │
│  (Browser)  │      │  (Port 5001) │
└─────────────┘      └───────┬──────┘
                             │
                             ▼
                      ┌─────────────┐
                      │  PostgreSQL │
                      │  (Port 5432)│
                      └──────▲──────┘
                             │
                      ┌──────┴──────┐
                      │   Scraper   │
                      │ (Run Once)  │
                      └─────────────┘
```

- **Server**: Reads data, serves HTTP API
- **Scraper**: Writes data, exits when done
- **PostgreSQL**: Shared data store
- Both services connect to DB independently
