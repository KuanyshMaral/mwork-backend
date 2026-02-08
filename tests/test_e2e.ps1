# MWork PhotoStudio Integration E2E Test
# Simple and reliable test script

$MWorkURL = "http://localhost:8080"
$PhotoStudioURL = "http://localhost:8090"

Write-Host "`n=== MWork + PhotoStudio E2E Test ===" -ForegroundColor Cyan
Write-Host "MWork: $MWorkURL" -ForegroundColor Gray
Write-Host "PhotoStudio: $PhotoStudioURL`n" -ForegroundColor Gray

try {
    # Step 1: Check servers
    Write-Host "Step 1: Checking servers..." -ForegroundColor Yellow
    
    try {
        $mworkHealth = Invoke-RestMethod -Uri "$MWorkURL/health" -TimeoutSec 3
        Write-Host "  OK - MWork running" -ForegroundColor Green
    } catch {
        Write-Host "  FAIL - MWork not running" -ForegroundColor Red
        exit 1
    }
    
    try {
        $photoHealth = Invoke-RestMethod -Uri "$PhotoStudioURL/healthz" -TimeoutSec 3
        Write-Host "  OK - PhotoStudio running" -ForegroundColor Green
    } catch {
        Write-Host "  FAIL - PhotoStudio not running" -ForegroundColor Red
        exit 1
    }

    # Step 2: Register user
    Write-Host "`nStep 2: Registering user..." -ForegroundColor Yellow
    
    $userId = Get-Random -Minimum 1000 -Maximum 9999
    $regEmail = "testuser_$userId@example.com"
    $regPassword = "TestPass123!"
    
    $regBody = @{
        email = $regEmail
        password = $regPassword
        full_name = "Test User"
        role = "model"
    } | ConvertTo-Json
    
    try {
        $regResponse = Invoke-RestMethod -Uri "$MWorkURL/api/v1/auth/register" `
            -Method Post `
            -Body $regBody `
            -ContentType "application/json" `
            -TimeoutSec 10
        
        Write-Host "  OK - User registered: $regEmail" -ForegroundColor Green
        $userId = $regResponse.data.user.id
        Write-Host "  User ID: $userId" -ForegroundColor Gray
    } catch {
        Write-Host "  FAIL - Registration failed" -ForegroundColor Red
        exit 1
    }

    # Step 3: Wait for sync
    Write-Host "`nStep 3: Waiting for PhotoStudio sync..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    Write-Host "  OK - Sync complete" -ForegroundColor Green

    # Step 4: Login
    Write-Host "`nStep 4: Logging in..." -ForegroundColor Yellow
    
    $loginBody = @{
        email = $regEmail
        password = $regPassword
    } | ConvertTo-Json
    
    try {
        $loginResponse = Invoke-RestMethod -Uri "$MWorkURL/api/v1/auth/login" `
            -Method Post `
            -Body $loginBody `
            -ContentType "application/json" `
            -TimeoutSec 10
        
        if ($loginResponse.data -and $loginResponse.data.tokens -and $loginResponse.data.tokens.access_token) {
            $accessToken = $loginResponse.data.tokens.access_token
            Write-Host "  OK - Logged in successfully" -ForegroundColor Green
            $tokenPreview = if ($accessToken.Length -gt 20) { $accessToken.Substring(0, 20) } else { $accessToken }
            Write-Host "  Token: $tokenPreview..." -ForegroundColor Gray
        } else {
            Write-Host "  FAIL - No token in response" -ForegroundColor Red
            Write-Host "  Response: $($loginResponse | ConvertTo-Json)" -ForegroundColor Yellow
            exit 1
        }
    } catch {
        Write-Host "  FAIL - Login failed: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }

    # Step 5: Get studios
    Write-Host "`nStep 5: Getting studios list..." -ForegroundColor Yellow
    
    $headers = @{
        "Authorization" = "Bearer $accessToken"
    }
    
    try {
        $studiosResponse = Invoke-RestMethod -Uri "$MWorkURL/api/v1/photostudio/studios" `
            -Method Get `
            -Headers $headers `
            -TimeoutSec 10
        
        $studioCount = if ($studiosResponse.data.studios) { $studiosResponse.data.studios.Count } else { 0 }
        Write-Host "  OK - Got $studioCount studios" -ForegroundColor Green
    } catch {
        Write-Host "  FAIL - Could not get studios" -ForegroundColor Red
        Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Yellow
    }

    # Step 6: Create booking
    Write-Host "`nStep 6: Creating booking..." -ForegroundColor Yellow
    
    # Use time far in the future to avoid conflicts
    $futureDate = (Get-Date).AddDays(30)
    $bookingStart = Get-Date -Year $futureDate.Year -Month $futureDate.Month -Day $futureDate.Day -Hour 14 -Minute 0 -Second 0
    $bookingEnd = $bookingStart.AddHours(1)
    
    $startTimeStr = $bookingStart.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    $endTimeStr = $bookingEnd.ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    
    $bookingBody = @{
        studio_id = 4
        room_id = 4
        start_time = $startTimeStr
        end_time = $endTimeStr
        notes = "E2E test booking"
    } | ConvertTo-Json
    
    try {
        $bookingResponse = Invoke-RestMethod -Uri "$MWorkURL/api/v1/photostudio/bookings" `
            -Method Post `
            -Headers $headers `
            -Body $bookingBody `
            -ContentType "application/json" `
            -TimeoutSec 10
        
        Write-Host "  OK - Booking created" -ForegroundColor Green
        Write-Host "  Booking ID: $($bookingResponse.data.booking_id)" -ForegroundColor Gray
    } catch {
        Write-Host "  FAIL - Booking creation failed" -ForegroundColor Red
        Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Yellow
        
        if ($_.Exception.Response) {
            $status = $_.Exception.Response.StatusCode.value__
            Write-Host "  HTTP Status: $status" -ForegroundColor Yellow
        }
    }

    # Success!
    Write-Host "`n=== TEST PASSED ===" -ForegroundColor Green
    Write-Host "Integration working correctly!" -ForegroundColor Green
    Write-Host "`n"

} catch {
    Write-Host "`n=== TEST FAILED ===" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "`n"
    exit 1
}
