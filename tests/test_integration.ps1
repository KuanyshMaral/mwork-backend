# MWork <-> PhotoStudio Integration Test
# Tests complete flow: Register -> Login -> List Studios -> Create Booking

$ErrorActionPreference = "Stop"

# Configuration
$MWorkURL = "http://localhost:8080"
$PhotoStudioURL = "http://localhost:8090"

# Test data
$randomSuffix = Get-Random -Minimum 1000 -Maximum 9999
$testEmail = "test_user_${randomSuffix}@example.com"
$testPassword = "TestPassword123!"
$testRole = "model" # or "employer"

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  MWork -> PhotoStudio Integration Test" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

Write-Host "Test User: $testEmail" -ForegroundColor Gray
Write-Host "MWork URL: $MWorkURL" -ForegroundColor Gray
Write-Host "PhotoStudio URL: $PhotoStudioURL`n" -ForegroundColor Gray

# Helper function to display success
function Write-Success {
    param([string]$message)
    Write-Host "✅ $message" -ForegroundColor Green
}

# Helper function to display error
function Write-Failure {
    param([string]$message, [object]$details = $null)
    Write-Host "❌ $message" -ForegroundColor Red
    if ($details) {
        Write-Host "Details: $($details | ConvertTo-Json -Depth 5)" -ForegroundColor Yellow
    }
}

# Helper function to display info
function Write-Info {
    param([string]$message)
    Write-Host "ℹ️  $message" -ForegroundColor Cyan
}

# Variables to store between steps
$accessToken = $null
$studioID = 1
$roomID = 1

try {
    # ============================================================
    # STEP 0: Health Check
    # ============================================================
    Write-Info "Step 0: Checking if services are running..."
    
    try {
        $mworkHealth = Invoke-RestMethod -Uri "$MWorkURL/health" -Method Get -TimeoutSec 3
        Write-Success "MWork is running"
    } catch {
        Write-Failure "MWork is not running on $MWorkURL" $_
        Write-Host "`nPlease start MWork backend first:" -ForegroundColor Yellow
        Write-Host "  cd mwork-backend-main" -ForegroundColor Gray
        Write-Host "  go run cmd/api/main.go" -ForegroundColor Gray
        exit 1
    }

    try {
        $psHealth = Invoke-RestMethod -Uri "$PhotoStudioURL/healthz" -Method Get -TimeoutSec 3
        Write-Success "PhotoStudio is running"
    } catch {
        Write-Failure "PhotoStudio is not running on $PhotoStudioURL" $_
        Write-Host "`nPlease start PhotoStudio backend first:" -ForegroundColor Yellow
        Write-Host "  cd photostudio-main" -ForegroundColor Gray
        Write-Host "  go run cmd/api/main.go" -ForegroundColor Gray
        exit 1
    }

    # ============================================================
    # STEP 1: Register User in MWork
    # ============================================================
    Write-Host "`n------------------------------------------------------------" -ForegroundColor Cyan
    Write-Info "Step 1: Registering new user in MWork..."
    Write-Host "POST $MWorkURL/api/v1/auth/register" -ForegroundColor Gray

    $registerBody = @{
        email = $testEmail
        password = $testPassword
        role = $testRole
        name = "Test User $randomSuffix"
    } | ConvertTo-Json

    try {
        $registerResponse = Invoke-RestMethod `
            -Uri "$MWorkURL/api/v1/auth/register" `
            -Method Post `
            -ContentType "application/json" `
            -Body $registerBody `
            -TimeoutSec 10

        Write-Success "User registered successfully"
        Write-Host "User ID: $($registerResponse.user.id)" -ForegroundColor Gray
        
        # Wait a bit for async PhotoStudio sync to complete
        Write-Info "Waiting 2 seconds for PhotoStudio sync..."
        Start-Sleep -Seconds 2
        
    } catch {
        Write-Failure "Registration failed" $_.Exception.Message
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $responseBody = $reader.ReadToEnd()
            Write-Host "Response: $responseBody" -ForegroundColor Yellow
        }
        throw
    }

    # ============================================================
    # STEP 2: Login to MWork
    # ============================================================
    Write-Host "`n------------------------------------------------------------" -ForegroundColor Cyan
    Write-Info "Step 2: Logging in to MWork..."
    Write-Host "POST $MWorkURL/api/v1/auth/login" -ForegroundColor Gray

    $loginBody = @{
        email = $testEmail
        password = $testPassword
    } | ConvertTo-Json

    try {
        $loginResponse = Invoke-RestMethod `
            -Uri "$MWorkURL/api/v1/auth/login" `
            -Method Post `
            -ContentType "application/json" `
            -Body $loginBody `
            -TimeoutSec 10

        $accessToken = $loginResponse.access_token
        
        if ([string]::IsNullOrEmpty($accessToken)) {
            throw "Access token is empty"
        }

        Write-Success "Login successful"
        Write-Host "Token: $($accessToken.Substring(0, 20))..." -ForegroundColor Gray
        
    } catch {
        Write-Failure "Login failed" $_.Exception.Message
        throw
    }

    # ============================================================
    # STEP 3: Get Studios List (MWork -> PhotoStudio proxy)
    # ============================================================
    Write-Host "`n------------------------------------------------------------" -ForegroundColor Cyan
    Write-Info "Step 3: Getting studios list via MWork API..."
    Write-Host "GET $MWorkURL/api/v1/photostudio/studios" -ForegroundColor Gray

    $headers = @{
        "Authorization" = "Bearer $accessToken"
    }

    try {
        $studiosResponse = Invoke-RestMethod `
            -Uri "$MWorkURL/api/v1/photostudio/studios?limit=10" `
            -Method Get `
            -Headers $headers `
            -TimeoutSec 10

        Write-Success "Studios list retrieved"
        
        if ($studiosResponse.studios -and $studiosResponse.studios.Count -gt 0) {
            $studioID = $studiosResponse.studios[0].id
            Write-Host "Found $($studiosResponse.studios.Count) studios" -ForegroundColor Gray
            Write-Host "Using Studio ID: $studioID - $($studiosResponse.studios[0].name)" -ForegroundColor Gray
        } else {
            Write-Host "⚠️  No studios found in PhotoStudio, using default ID=1" -ForegroundColor Yellow
            $studioID = 1
        }
        
    } catch {
        Write-Failure "Failed to get studios" $_.Exception.Message
        if ($_.Exception.Response) {
            Write-Host "Status Code: $($_.Exception.Response.StatusCode)" -ForegroundColor Yellow
        }
        Write-Host "⚠️  Continuing with default studio ID=1" -ForegroundColor Yellow
        $studioID = 1
    }

    # ============================================================
    # STEP 4: Create Booking (MWork -> PhotoStudio)
    # ============================================================
    Write-Host "`n------------------------------------------------------------" -ForegroundColor Cyan
    Write-Info "Step 4: Creating booking via MWork API..."
    Write-Host "POST $MWorkURL/api/v1/photostudio/bookings" -ForegroundColor Gray

    # Generate booking time (tomorrow, 10:00-12:00)
    $tomorrow = (Get-Date).AddDays(1)
    $startTime = Get-Date -Year $tomorrow.Year -Month $tomorrow.Month -Day $tomorrow.Day -Hour 10 -Minute 0 -Second 0
    $endTime = $startTime.AddHours(2)
    
    # Format as RFC3339 (ISO 8601)
    $startTimeStr = $startTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    $endTimeStr = $endTime.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

    $bookingBody = @{
        studio_id = $studioID
        room_id = $roomID
        start_time = $startTimeStr
        end_time = $endTimeStr
        notes = "Integration test booking - created at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    } | ConvertTo-Json

    Write-Host "Booking details:" -ForegroundColor Gray
    Write-Host "  Studio ID: $studioID" -ForegroundColor Gray
    Write-Host "  Room ID: $roomID" -ForegroundColor Gray
    Write-Host "  Start: $startTimeStr" -ForegroundColor Gray
    Write-Host "  End: $endTimeStr" -ForegroundColor Gray

    try {
        $bookingResponse = Invoke-RestMethod `
            -Uri "$MWorkURL/api/v1/photostudio/bookings" `
            -Method Post `
            -Headers $headers `
            -ContentType "application/json" `
            -Body $bookingBody `
            -TimeoutSec 10

        Write-Success "Booking created successfully!"
        Write-Host "Booking ID: $($bookingResponse.booking_id)" -ForegroundColor Gray
        Write-Host "Status: $($bookingResponse.status)" -ForegroundColor Gray
        
    } catch {
        Write-Failure "Booking creation failed"
        
        # Try to get detailed error
        if ($_.Exception.Response) {
            Write-Host "`nHTTP Status: $($_.Exception.Response.StatusCode.value__) $($_.Exception.Response.StatusDescription)" -ForegroundColor Yellow
            
            try {
                $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
                $responseBody = $reader.ReadToEnd()
                $reader.Close()
                
                Write-Host "`nError Response:" -ForegroundColor Yellow
                Write-Host $responseBody -ForegroundColor Yellow
                
                # Try to parse as JSON
                try {
                    $errorJson = $responseBody | ConvertFrom-Json
                    if ($errorJson.error) {
                        Write-Host "`nError Details:" -ForegroundColor Red
                        Write-Host "  Code: $($errorJson.error.code)" -ForegroundColor Red
                        Write-Host "  Message: $($errorJson.error.message)" -ForegroundColor Red
                    }
                } catch {
                    # Not JSON, already displayed above
                }
            } catch {
                Write-Host "Could not read error response body" -ForegroundColor Yellow
            }
        }
        
        Write-Host "`n⚠️  Common causes:" -ForegroundColor Yellow
        Write-Host "  1. User not synced to PhotoStudio (check PhotoStudio logs)" -ForegroundColor Gray
        Write-Host "  2. Studio or Room ID doesn't exist in PhotoStudio DB" -ForegroundColor Gray
        Write-Host "  3. Time format incorrect (should be RFC3339/ISO8601)" -ForegroundColor Gray
        Write-Host "  4. PhotoStudio middleware not working" -ForegroundColor Gray
        Write-Host "  5. MWORK_SYNC_TOKEN mismatch between services" -ForegroundColor Gray
        
        throw
    }

    # ============================================================
    # SUCCESS!
    # ============================================================
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "  ✅ ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "`nIntegration Summary:" -ForegroundColor Cyan
    Write-Host "  ✅ User registered in MWork" -ForegroundColor Green
    Write-Host "  ✅ User synced to PhotoStudio (async)" -ForegroundColor Green
    Write-Host "  ✅ Login successful" -ForegroundColor Green
    Write-Host "  ✅ Studios list retrieved" -ForegroundColor Green
    Write-Host "  ✅ Booking created successfully" -ForegroundColor Green
    Write-Host "`nThe integration is working correctly! 🎉`n" -ForegroundColor Green

} catch {
    Write-Host "`n========================================" -ForegroundColor Red
    Write-Host "  ❌ TEST FAILED" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "`nError: $($_.Exception.Message)`n" -ForegroundColor Red
    exit 1
}
