# Verify Rolling Logs

$logDir = "logs"

# Clean up existing logs
if (Test-Path $logDir) {
    Remove-Item "$logDir\*.log" -Force -ErrorAction Stop
}

# Run the app 12 times
for ($i = 1; $i -le 12; $i++) {
    Write-Host "Run #$i"
    $process = Start-Process -FilePath ".\app.exe" -NoNewWindow -PassThru
    Start-Sleep -Seconds 3
    if (!$process.HasExited) {
        Stop-Process -Id $process.Id -Force
    }
    Start-Sleep -Seconds 1
}

# Check logs
$logs = Get-ChildItem "$logDir\*.log"
Write-Host "Total log files: $($logs.Count)"

if ($logs.Count -eq 10) {
    Write-Host "SUCCESS: Found 10 log files."
} else {
    Write-Host "FAILURE: Found $($logs.Count) log files. Expected 10."
}

# List files to verify timestamps
$logs | Sort-Object Name | ForEach-Object { Write-Host $_.Name }
