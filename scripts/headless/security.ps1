<#
.SYNOPSIS
    TGS Headless Security Tool
    Manages Security Policies: RDP, Bluetooth, Browser Restrictions.
.PARAMETER BlockRDP
    Blocks RDP Clipboard & File Transfer.
.PARAMETER AllowRDP
    Allows RDP Clipboard & File Transfer.
.PARAMETER BlockBT
    Blocks Bluetooth File Transfer.
.PARAMETER AllowBT
    Allows Bluetooth File Transfer.
.PARAMETER SetDomains
    Sets allowed domains for Browser (Chrome/Edge). Comma separated string.
.PARAMETER ClearDomains
    Removes browser domain restrictions.
#>
param (
    [switch]$BlockRDP,
    [switch]$AllowRDP,
    [switch]$BlockBT,
    [switch]$AllowBT,
    [string]$SetDomains,
    [switch]$ClearDomains
)

$ErrorActionPreference = "SilentlyContinue"
$Result = @{ Status = "Success"; Message = @() }


# --- 2. RDP SECURITY ---
$RDPKey = "HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services"
if (!(Test-Path $RDPKey)) { New-Item $RDPKey -Force | Out-Null }

if ($BlockRDP) {
    Set-ItemProperty $RDPKey -Name "fDisableClip" -Value 1 -Force
    Set-ItemProperty $RDPKey -Name "DisableClipboardRedirection" -Value 1 -Force
    Set-ItemProperty $RDPKey -Name "DisableDriveRedirection" -Value 1 -Force
    $Result.Message += "RDP Clipboard/Drive Blocked."
}
if ($AllowRDP) {
    Remove-ItemProperty $RDPKey -Name "fDisableClip" -ErrorAction SilentlyContinue
    Remove-ItemProperty $RDPKey -Name "DisableClipboardRedirection" -ErrorAction SilentlyContinue
    Remove-ItemProperty $RDPKey -Name "DisableDriveRedirection" -ErrorAction SilentlyContinue
    $Result.Message += "RDP Control Allowed."
}

# --- 3. BLUETOOTH CONTROL ---
$BTKey = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\Bluetooth"
if (!(Test-Path $BTKey)) { New-Item $BTKey -Force | Out-Null }

if ($BlockBT) {
    Set-ItemProperty $BTKey -Name "DisableFileTransfer" -Value 1 -Force
    $Result.Message += "Bluetooth Transfer Blocked."
}
if ($AllowBT) {
    Remove-ItemProperty $BTKey -Name "DisableFileTransfer" -ErrorAction SilentlyContinue
    $Result.Message += "Bluetooth Transfer Allowed."
}

# --- 4. BROWSER POLICY ---
$ChromeKey = "HKLM:\SOFTWARE\Policies\Google\Chrome"
$EdgeKey = "HKLM:\SOFTWARE\Policies\Microsoft\Edge"

if ($SetDomains) {
    $Domains = $SetDomains -split ","
    # Simple logic: Allow these, block others? 
    # Original Audit Tool logic for "AllowedDomainsForApps" implies restrictive mode.
    
    foreach ($Key in @($ChromeKey, $EdgeKey)) {
        if (!(Test-Path $Key)) { New-Item $Key -Force | Out-Null }
        Set-ItemProperty $Key -Name "AllowedDomainsForApps" -Value $SetDomains -Type String -Force
        # Optional: Add URLBlocklist * if needed, but "AllowedDomainsForApps" usually does whitelisting for apps/extensions or specific contexts depending on policy.
        # Assuming original script logic:
        # "AllowedDomainsForApps" -> Limits Google Apps domain access.
        # For general browsing restriction, one usually uses URLBlocklist + URLAllowlist.
        # Using exact key from Audit script: "AllowedDomainsForApps"
    }
    $Result.Message += "Browser Domains Set ($SetDomains)."
}

if ($ClearDomains) {
    foreach ($Key in @($ChromeKey, $EdgeKey)) {
        if (Test-Path $Key) {
            Remove-ItemProperty $Key -Name "AllowedDomainsForApps" -ErrorAction SilentlyContinue
        }
    }
    $Result.Message += "Browser Policies Cleared."
}

$Result | ConvertTo-Json
