# MWork API - PowerShell Commands
# Usage: .\scripts\dev.ps1 <command>

param(
    [Parameter(Position=0)]
    [string]$Command = "help"
)

$DatabaseUrl = "postgresql://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable"

function Show-Help {
    Write-Host "MWork API - Available Commands:" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  dev          - Run with hot reload (requires air)" -ForegroundColor Green
    Write-Host "  run          - Run the application" -ForegroundColor Green
    Write-Host "  build        - Build the binary" -ForegroundColor Green
    Write-Host "  test         - Run tests" -ForegroundColor Green
    Write-Host ""
    Write-Host "  docker-up    - Start Docker containers" -ForegroundColor Yellow
    Write-Host "  docker-down  - Stop Docker containers" -ForegroundColor Yellow
    Write-Host "  docker-logs  - View Docker logs" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  migrate-up   - Apply all migrations" -ForegroundColor Magenta
    Write-Host "  migrate-down - Rollback last migration" -ForegroundColor Magenta
    Write-Host ""
    Write-Host "  deps         - Download dependencies" -ForegroundColor Blue
    Write-Host "  tidy         - Tidy dependencies" -ForegroundColor Blue
    Write-Host ""
}

function Start-Dev {
    Write-Host "Starting development server with hot reload..." -ForegroundColor Green
    air
}

function Start-Run {
    Write-Host "Running application..." -ForegroundColor Green
    go run ./cmd/api/main.go
}

function Start-Build {
    Write-Host "Building..." -ForegroundColor Green
    if (-not (Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin" | Out-Null
    }
    go build -o bin/mwork-api.exe ./cmd/api/main.go
    Write-Host "Built: bin/mwork-api.exe" -ForegroundColor Green
}

function Start-Test {
    Write-Host "Running tests..." -ForegroundColor Green
    go test -v ./...
}

function Start-DockerUp {
    Write-Host "Starting Docker containers..." -ForegroundColor Yellow
    docker-compose -f docker/docker-compose.yml up -d
}

function Start-DockerDown {
    Write-Host "Stopping Docker containers..." -ForegroundColor Yellow
    docker-compose -f docker/docker-compose.yml down
}

function Start-DockerLogs {
    docker-compose -f docker/docker-compose.yml logs -f
}

function Start-MigrateUp {
    Write-Host "Applying migrations..." -ForegroundColor Magenta
    migrate -path migrations -database $DatabaseUrl up
}

function Start-MigrateDown {
    Write-Host "Rolling back last migration..." -ForegroundColor Magenta
    migrate -path migrations -database $DatabaseUrl down 1
}

function Start-Deps {
    Write-Host "Downloading dependencies..." -ForegroundColor Blue
    go mod download
}

function Start-Tidy {
    Write-Host "Tidying dependencies..." -ForegroundColor Blue
    go mod tidy
}

# Execute command
switch ($Command.ToLower()) {
    "help"        { Show-Help }
    "dev"         { Start-Dev }
    "run"         { Start-Run }
    "build"       { Start-Build }
    "test"        { Start-Test }
    "docker-up"   { Start-DockerUp }
    "docker-down" { Start-DockerDown }
    "docker-logs" { Start-DockerLogs }
    "migrate-up"  { Start-MigrateUp }
    "migrate-down"{ Start-MigrateDown }
    "deps"        { Start-Deps }
    "tidy"        { Start-Tidy }
    default       { 
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Show-Help 
    }
}
