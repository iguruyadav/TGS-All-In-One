<#
.SYNOPSIS
    TGS Headless Setup Tool — Full Config Script
#>
param (
    [switch]$GetStatus,
    [switch]$SetTime,
    [string]$SetName,
    [switch]$EnableRDP,
    [switch]$DisableRDP,
    [switch]$Map121,
    [switch]$Remove121,
    [string]$MapNAS,      # format: "username|password"
    [switch]$RemoveNAS,
    [switch]$SetVisuals,
    [switch]$AddThisPcIcon,
    [switch]$SetWallpaper,
    [switch]$UpgradeWindows,
    [switch]$Reboot
)

$ErrorActionPreference = "SilentlyContinue"
$Result = @{ Status = "Success"; Message = @(); RebootRequired = $false }

# --- Helpers ---
function Get-RDPStatus {
    $v = Get-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server" -Name "fDenyTSConnections"
    if ($null -eq $v) { return "Unknown" }
    return if ($v.fDenyTSConnections -eq 0) { "Enabled" } else { "Disabled" }
}
function Get-DriveStatus($D) { if (Test-Path "$D`:") { "Connected" } else { "Disconnected" } }

# --- Asset paths (relative to exe location) ---
$AppDir = Split-Path -Parent $MyInvocation.MyCommand.Path
# Try to find bg.jpg in common locations
$WallPath = Join-Path $AppDir "frontend\src\assets\images\bg.jpg"
if (-not (Test-Path $WallPath)) {
    $WallPath = Join-Path $AppDir "assets\images\bg.jpg"
}

# ─── 0. STATUS ─────────────────────────────────────────────────
if ($GetStatus) {
    @{
        PCName     = $env:COMPUTERNAME
        RDPStatus  = Get-RDPStatus
        Server121  = Get-DriveStatus "Z"
        NASStorage = Get-DriveStatus "Y"
        TimeZone   = (Get-TimeZone).DisplayName
    } | ConvertTo-Json
    exit
}

# ─── 1. TIMEZONE ───────────────────────────────────────────────
if ($SetTime) {
    tzutil /s "India Standard Time" | Out-Null
    Start-Process "w32tm" -ArgumentList "/resync" -NoNewWindow -Wait -ErrorAction SilentlyContinue
    $Result.Message += "Timezone set to IST."
}

# ─── 2. RENAME PC ─────────────────────────────────────────────
if ($SetName) {
    if ($SetName -ne $env:COMPUTERNAME) {
        Rename-Computer -NewName $SetName -Force -ErrorAction Stop
        $Result.Message += "PC renamed to $SetName."
        $Result.RebootRequired = $true
    }
    else {
        $Result.Message += "Name unchanged."
    }
}

# ─── 3. RDP ───────────────────────────────────────────────────
if ($EnableRDP) {
    Set-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server" "fDenyTSConnections" 0 -Force
    Enable-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue
    $Result.Message += "RDP Enabled."
}
if ($DisableRDP) {
    Set-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server" "fDenyTSConnections" 1 -Force
    $Result.Message += "RDP Disabled."
}

# ─── 4. THIIS PC ICON ─────────────────────────────────────────
if ($AddThisPcIcon) {
    $Key = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel"
    $CLSID = "{20D04FE0-3AEA-1069-A2D8-08002B30309D}"
    if (-not (Test-Path $Key)) { New-Item -Path $Key -Force | Out-Null }
    Set-ItemProperty -Path $Key -Name $CLSID -Value 0 -Type DWord -Force | Out-Null
    $Result.Message += "This PC icon added to Desktop."
}

# ─── 5. WALLPAPER ─────────────────────────────────────────────
if ($SetWallpaper) {
    if (Test-Path $WallPath) {
        Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class Wallpaper {
    [DllImport("user32.dll", CharSet=CharSet.Auto)]
    public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
}
"@ -ErrorAction SilentlyContinue
        $AbsWall = (Resolve-Path $WallPath).Path
        [Wallpaper]::SystemParametersInfo(0x0014, 0, $AbsWall, 0x1 -bor 0x2) | Out-Null
        $Result.Message += "Wallpaper set."
    }
    else {
        $Result.Message += "Wallpaper file not found: $WallPath"
    }
}

# ─── 6. NETWORK DRIVES ────────────────────────────────────────
if ($Map121) {
    net use Z: /delete /y 2>&1 | Out-Null
    cmdkey /add:174.156.5.121 /user:tgsuser121 "/pass:@121#Sgt" | Out-Null
    net use Z: \\174.156.5.121\Share /user:tgsuser121 "@121#Sgt" /persistent:yes 2>&1 | Out-Null
    $Result.Message += "Z: drive connected."
}
if ($Remove121) {
    net use Z: /delete /y 2>&1 | Out-Null
    $Result.Message += "Z: drive removed."
}
if ($MapNAS) {
    $parts = $MapNAS -split "\|", 2
    $nasU = $parts[0]
    $nasP = if ($parts.Count -gt 1) { $parts[1] } else { "" }
    net use Y: /delete /y 2>&1 | Out-Null
    cmdkey /add:174.156.4.3 /user:$nasU /pass:$nasP | Out-Null
    net use Y: \\174.156.4.3\NAS /user:$nasU $nasP /persistent:yes 2>&1 | Out-Null
    $Result.Message += "Y: NAS connected."
}
if ($RemoveNAS) {
    net use Y: /delete /y 2>&1 | Out-Null
    $Result.Message += "Y: drive removed."
}

# ─── 7. WINDOWS UPGRADE ───────────────────────────────────────
if ($UpgradeWindows) {
    $Key = "VK7JG-NPHTM-C97JM-9MPGT-3V66T"
    Start-Process "cscript" -ArgumentList "//B C:\Windows\System32\slmgr.vbs /ipk $Key" -NoNewWindow -Wait
    Start-Process "cscript" -ArgumentList "//B C:\Windows\System32\slmgr.vbs /ato" -NoNewWindow -Wait
    $Result.Message += "Windows upgrade key applied."
}

# ─── 8. REBOOT ────────────────────────────────────────────────
if ($Reboot) {
    Restart-Computer -Force
}

$Result | ConvertTo-Json
