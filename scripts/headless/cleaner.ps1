<#
.SYNOPSIS
    TGS Headless Cleaner Tool
    Performs system cleanup operations.
.PARAMETER CleanUpdates
    Clears Windows Update Cache (Stops wuauserv).
.PARAMETER CleanTemp
    Clears Temp folders, Prefetch, WER, Shader Cache.
.PARAMETER CleanSxS
    Runs DISM Component Cleanup (Slow).
.PARAMETER CleanBin
    Empties Recycle Bin and Flushes DNS.
.PARAMETER CleanVSS
    Resizes ShadowStorage to 5GB.
.PARAMETER RemoveBloat
    Removes common bloatware (Xbox, Solitaire, Bing, etc.).
.PARAMETER DeepClean
    Runs ALL cleanup tasks.
#>
param (
    [switch]$CleanUpdates,
    [switch]$CleanTemp,
    [switch]$CleanSxS,
    [switch]$CleanBin,
    [switch]$CleanVSS,
    [switch]$RemoveBloat,
    [switch]$DeepClean
)

$ErrorActionPreference = "SilentlyContinue"
$Result = @{ Status = "Success"; Message = @() }

if ($DeepClean) {
    $CleanUpdates = $true; $CleanTemp = $true; $CleanSxS = $true; 
    $CleanBin = $true; $CleanVSS = $true; $RemoveBloat = $true
}

# Helper to remove files
function Safe-Clean ($Path) {
    if (Test-Path $Path) {
        Remove-Item -Path "$Path\*" -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# 1. WINDOWS UPDATES
if ($CleanUpdates) {
    try {
        Stop-Service wuauserv -Force -ErrorAction SilentlyContinue
        Safe-Clean "C:\Windows\SoftwareDistribution\Download"
        Start-Service wuauserv
        $Result.Message += "Update Cache Cleared."
    }
    catch { $Result.Message += "Update Clean Error: $_" }
}

# 2. TEMP & CACHES
if ($CleanTemp) {
    try {
        Safe-Clean "C:\Windows\Temp"
        Safe-Clean "$env:TEMP"
        # Prefetch (Keep Layout.ini)
        Get-ChildItem "C:\Windows\Prefetch" -ErrorAction SilentlyContinue | Where-Object { $_.Name -ne "Layout.ini" } | Remove-Item -Force -ErrorAction SilentlyContinue
        # WER
        Remove-Item "C:\ProgramData\Microsoft\Windows\WER\*" -Recurse -Force -ErrorAction SilentlyContinue
        # Shader Cache
        Remove-Item "$env:LOCALAPPDATA\D3DSCache\*" -Recurse -Force -ErrorAction SilentlyContinue
        
        $Result.Message += "Temp & Caches Purged."
    }
    catch { $Result.Message += "Temp Clean Error: $_" }
}

# 3. WINSXS
if ($CleanSxS) {
    try {
        Start-Process -FilePath "dism.exe" -ArgumentList "/online /Cleanup-Image /StartComponentCleanup" -Wait -WindowStyle Hidden
        $Result.Message += "WinSxS Optimized."
    }
    catch { $Result.Message += "WinSxS Error: $_" }
}

# 4. RECYCLE BIN & DNS
if ($CleanBin) {
    try {
        Clear-RecycleBin -Force -ErrorAction SilentlyContinue
        Clear-DnsClientCache -ErrorAction SilentlyContinue
        $Result.Message += "Bin Emptied & DNS Flushed."
    }
    catch { $Result.Message += "Bin/DNS Error: $_" }
}

# 5. VSS RESIZE
if ($CleanVSS) {
    try {
        vssadmin Resize ShadowStorage /For=C: /On=C: /MaxSize=5GB | Out-Null
        $Result.Message += "VSS Resized to 5GB."
    }
    catch { $Result.Message += "VSS Error: $_" }
}

# 6. BLOATWARE REMOVAL
if ($RemoveBloat) {
    $BloatList = @("*Xbox*", "*Solitaire*", "*BingWeather*", "*GetHelp*", "*FeedbackHub*", "*Zune*", "*People*", "*StickyNotes*", "*3DViewer*", "*Groove*", "*MixedReality*", "*CandyCrush*", "*Disney*", "*Spotify*", "*Netflix*")
    $Count = 0
    foreach ($App in $BloatList) {
        $Pkg = Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -like $App }
        if ($Pkg) {
            Remove-AppxProvisionedPackage -Online -PackageName $Pkg.PackageName -ErrorAction SilentlyContinue | Out-Null
            $Count++
        }
    }
    $Result.Message += "Removed $Count Bloatware Apps."
}

$Result | ConvertTo-Json
