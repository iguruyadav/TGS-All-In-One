# USB unblock fix
Write-Host "Removing removable storage block policies..." -ForegroundColor Cyan

$paths = @(
    "HKLM:\SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices",
    "HKCU:\SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices"
)

foreach ($path in $paths) {
    if (Test-Path $path) {
        Remove-ItemProperty -Path $path -Name "Deny_All" -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path $path -Name "Deny_Read" -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path $path -Name "Deny_Write" -ErrorAction SilentlyContinue
    }
}

Write-Host "Setting USBSTOR to default..." -ForegroundColor Cyan
reg add "HKLM\SYSTEM\CurrentControlSet\Services\USBSTOR" /v Start /t REG_DWORD /d 3 /f | Out-Null

Write-Host "Restarting shell and rescanning disks..." -ForegroundColor Cyan
Get-Process explorer -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Process explorer.exe
Start-Sleep -Seconds 2

Write-Host "Done. Please unplug and replug the USB device." -ForegroundColor Green