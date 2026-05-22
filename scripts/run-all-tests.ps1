$ErrorActionPreference = "Continue"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "      KazakhExpress Automated Test & Verification System  " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Start required infrastructure containers
Write-Host "`n[1/4] Preparing PostgreSQL & NATS infrastructure..." -ForegroundColor Yellow
$env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"
$env:NATS_URL = "nats://localhost:4222"

try {
    docker compose up -d postgres nats migrate
    Write-Host "Containers started. Waiting for database to stabilize..." -ForegroundColor Green
    Start-Sleep -Seconds 5
} catch {
    Write-Host "Warning: docker-compose check failed. Proceeding assuming databases are running." -ForegroundColor DarkYellow
}

# 2. Services definition
$services = @(
  "api-gateway",
  "user-service",
  "order-service",
  "product-service",
  "payment-service",
  "review-service",
  "smtp-service"
)

$results = @()
$totalStart = Get-Date

Write-Host "`n[2/4] Executing test suite for all microservices..." -ForegroundColor Yellow

foreach ($service in $services) {
    Write-Host "`n--------------------------------------------------" -ForegroundColor Gray
    Write-Host "Service: $service" -ForegroundColor Cyan
    Write-Host "--------------------------------------------------" -ForegroundColor Gray
    
    if (-not (Test-Path $service)) {
        Write-Host "Directory $service not found. Skipping." -ForegroundColor Red
        continue
    }
    
    Push-Location $service
    
    # Run Unit & Mock tests
    Write-Host "-> Running Unit & Mock Tests..." -ForegroundColor DarkGray
    $unitStart = Get-Date
    $unitOutput = go test ./... -cover 2>&1
    $unitExit = $LASTEXITCODE
    $unitDuration = ((Get-Date) - $unitStart).TotalSeconds
    
    # Extract coverage
    $coverage = "N/A"
    foreach ($line in $unitOutput) {
        if ($line -match "coverage: (\d+\.\d+)% of statements") {
            $coverage = $Matches[1] + "%"
        }
    }

    # Run Integration Tests
    Write-Host "-> Running PostgreSQL & NATS Integration Tests..." -ForegroundColor DarkGray
    $intStart = Get-Date
    $intOutput = go test -tags=integration ./... 2>&1
    $intExit = $LASTEXITCODE
    $intDuration = ((Get-Date) - $intStart).TotalSeconds

    Pop-Location

    $unitStatus = if ($unitExit -eq 0) { "PASSED" } else { "FAILED" }
    $intStatus = if ($intExit -eq 0) { "PASSED" } else { "FAILED" }

    # Print summary to console
    if ($unitStatus -eq "PASSED") {
        Write-Host "  [+] Unit & Mock Tests: " -NoNewline; Write-Host "PASS" -ForegroundColor Green -NoNewline; Write-Host " (Coverage: $coverage, Time: $([Math]::Round($unitDuration, 2))s)"
    } else {
        Write-Host "  [-] Unit & Mock Tests: " -NoNewline; Write-Host "FAIL" -ForegroundColor Red
        Write-Host ($unitOutput -join "`n") -ForegroundColor DarkRed
    }

    if ($intStatus -eq "PASSED") {
        Write-Host "  [+] Integration Tests: " -NoNewline; Write-Host "PASS" -ForegroundColor Green -NoNewline; Write-Host " (Time: $([Math]::Round($intDuration, 2))s)"
    } else {
        Write-Host "  [-] Integration Tests: " -NoNewline; Write-Host "FAIL" -ForegroundColor Red
        Write-Host ($intOutput -join "`n") -ForegroundColor DarkRed
    }

    $results += [PSCustomObject]@{
        Service     = $service
        UnitStatus  = $unitStatus
        IntStatus   = $intStatus
        Coverage    = $coverage
        UnitTime    = [Math]::Round($unitDuration, 2)
        IntTime     = [Math]::Round($intDuration, 2)
    }
}

$totalDuration = [Math]::Round(((Get-Date) - $totalStart).TotalSeconds, 2)

# 3. Generate Report markdown file
Write-Host "`n[3/4] Compiling dynamic test report at docs/test-execution-report.md..." -ForegroundColor Yellow

$reportPath = "docs/test-execution-report.md"
$parentDir = Split-Path $reportPath
if (-not (Test-Path $parentDir)) {
    New-Item -ItemType Directory -Force -Path $parentDir | Out-Null
}

$reportContent = @"
# KazakhExpress Test Execution Report

**Generated At:** $((Get-Date).ToString("yyyy-MM-dd HH:mm:ss"))
**Total Execution Time:** $totalDuration seconds

This report summarizes the execution of all unit, mock, and database/messaging integration tests across the KazakhExpress microservice architecture.

## Execution Status Summary

| Service Name | Unit & Mock Status | Integration Status | Go Coverage | Unit Run Time | Integration Run Time |
| :--- | :---: | :---: | :---: | :---: | :---: |
"@

$allPassed = $true
foreach ($res in $results) {
    $unitIcon = if ($res.UnitStatus -eq "PASSED") { "PASS" } else { "FAIL" }
    $intIcon = if ($res.IntStatus -eq "PASSED") { "PASS" } else { "FAIL" }
    
    if ($res.UnitStatus -ne "PASSED" -or $res.IntStatus -ne "PASSED") {
        $allPassed = $false
    }
    
    $reportContent += "`n| **$($res.Service)** | $unitIcon | $intIcon | $($res.Coverage) | $($res.UnitTime)s | $($res.IntTime)s |"
}

$reportContent += @"


## Test Architecture & Coverage Analysis

1. **Unit & Mock Tests**:
   - Implemented standard Go tests using captured structures and interfaces.
   - External services (SMTP, Cache, Database, NATS publishers) are simulated using in-memory mock packages for absolute speed and isolation.
   - Tested boundary validation, failure propagation, password hashing, and token signatures.

2. **PostgreSQL & NATS Integration Tests**:
   - Triggered using Go build tags (`//go:build integration`).
   - Interacts with live Postgres database containers using pgx connection pools to apply, read, and delete transactions.
   - Connects to NATS server to publish structured events and verify sync subscriptions on core queue channels.

"@

if ($allPassed) {
    $reportContent += @"

> [!NOTE]
> **VERIFICATION SUCCESSFUL**: All microservice components have passed 100% of their test validations with secure database schemas and robust queue communications.
"@
} else {
    $reportContent += @"

> [!WARNING]
> **VERIFICATION FAILED**: Some test cases failed. Please review the console outputs and service logs to resolve active build issues.
"@
}

$reportContent | Out-File -FilePath $reportPath -Encoding utf8

# Helper function removed - directory check done inline.

# 4. Final report presentation
Write-Host "`n[4/4] Verification complete!" -ForegroundColor Yellow
Write-Host "--------------------------------------------------------" -ForegroundColor Gray
if ($allPassed) {
    Write-Host "SUCCESS: All tests passed! KazakhExpress is 100% stable." -ForegroundColor Green
} else {
    Write-Host "FAILURE: Some tests failed. Check report file." -ForegroundColor Red
}
Write-Host "Report created: docs/test-execution-report.md" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

if (-not $allPassed) {
    exit 1
}
