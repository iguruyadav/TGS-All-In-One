# USB Block Diagnostic Script
# Purpose: Scan common Windows settings that can block USB storage/devices.
# This script does NOT make changes. It only reports likely root causes.

# Run with: PowerShell (Run as Administrator) -> .\usb_block_diagnostic.ps1

$ErrorActionPreference = 'SilentlyContinue'
$report = New-Object System.Collections.Generic.List[object]

function Add-Report {
    param(
        [string]$Category,
        [string]$Check,
        [string]$Status,
        [string]$Details,
        [string]$Risk = ''
    )
    $report.Add([pscustomobject]@{
        Time     = (Get-Date).ToString('yyyy-MM-dd HH:mm:ss')
        Category = $Category
        Check    = $Check
        Status   = $Status
        Details  = $Details
        Risk     = $Risk
    }) | Out-Null
}

function Get-RegValue {
    param([string]$Path,[string]$Name)
    try {
        $item = Get-ItemProperty -Path $Path -ErrorAction Stop
        return $item.$Name
    } catch {
        return $null
    }
}

function Test-Admin {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Admin)) {
    Add-Report -Category 'Session' -Check 'Administrator rights' -Status 'Warning' -Details 'Script is not running as Administrator. Some checks may be incomplete.' -Risk 'Medium'
} else {
    Add-Report -Category 'Session' -Check 'Administrator rights' -Status 'OK' -Details 'Script is running with Administrator rights.' -Risk 'Low'
}

# 1) USBSTOR service
$usbStorStart = Get-RegValue 'HKLM:\SYSTEM\CurrentControlSet\Services\USBSTOR' 'Start'
$usbStorMap = @{0='Boot';1='System';2='Automatic';3='Manual';4='Disabled'}
if ($null -eq $usbStorStart) {
    Add-Report 'Storage Driver' 'USBSTOR service' 'Not Found' 'USBSTOR registry key not found.' 'High'
} elseif ($usbStorStart -eq 4) {
    Add-Report 'Storage Driver' 'USBSTOR service' 'Blocked' "USBSTOR Start=$usbStorStart ($($usbStorMap[$usbStorStart])). USB storage is disabled." 'High'
} else {
    Add-Report 'Storage Driver' 'USBSTOR service' 'OK' "USBSTOR Start=$usbStorStart ($($usbStorMap[$usbStorStart]))." 'Low'
}

# 2) Removable Storage Access policies
$removableBase = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices'
$denyAll = Get-RegValue $removableBase 'Deny_All'
$allowRemote = Get-RegValue $removableBase 'AllowRemoteDASD'
if ($denyAll -eq 1) {
    Add-Report 'Group Policy' 'Removable Storage Deny_All' 'Blocked' 'Policy Deny_All=1. All removable storage access is blocked.' 'High'
} elseif ($null -ne (Get-Item $removableBase)) {
    Add-Report 'Group Policy' 'Removable Storage policies' 'Review' "Policy key exists. Deny_All=$denyAll, AllowRemoteDASD=$allowRemote." 'Medium'
} else {
    Add-Report 'Group Policy' 'Removable Storage policies' 'OK' 'No machine-level RemovableStorageDevices policy key found.' 'Low'
}

# Common class GUIDs for removable devices
$classGuids = @(
    '{53f56307-b6bf-11d0-94f2-00a0c91efb8b}', # Disk drive/storage volumes
    '{4d36e967-e325-11ce-bfc1-08002be10318}', # Disk drives
    '{36fc9e60-c465-11cf-8056-444553540000}'  # USB controllers
)
foreach ($guid in $classGuids) {
    $path = Join-Path $removableBase $guid
    $readDenied = Get-RegValue $path 'Deny_Read'
    $writeDenied = Get-RegValue $path 'Deny_Write'
    $executeDenied = Get-RegValue $path 'Deny_Execute'
    if ($readDenied -eq 1 -or $writeDenied -eq 1 -or $executeDenied -eq 1) {
        Add-Report 'Group Policy' "Removable class $guid" 'Blocked' "Deny_Read=$readDenied, Deny_Write=$writeDenied, Deny_Execute=$executeDenied." 'High'
    }
}

# 3) Device installation restrictions
$devInstallBase = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DeviceInstall\Restrictions'
$denyRemovable = Get-RegValue $devInstallBase 'DenyRemovableDevices'
$denyUnspecified = Get-RegValue $devInstallBase 'DenyUnspecified'
$allowAdmins = Get-RegValue $devInstallBase 'AllowAdminInstall'
if ($denyRemovable -eq 1 -or $denyUnspecified -eq 1) {
    Add-Report 'Device Install' 'Installation restrictions' 'Blocked' "DenyRemovableDevices=$denyRemovable, DenyUnspecified=$denyUnspecified, AllowAdminInstall=$allowAdmins." 'High'
} elseif ($null -ne (Get-Item $devInstallBase)) {
    Add-Report 'Device Install' 'Installation restrictions' 'Review' "Restrictions key exists. DenyRemovableDevices=$denyRemovable, DenyUnspecified=$denyUnspecified, AllowAdminInstall=$allowAdmins." 'Medium'
} else {
    Add-Report 'Device Install' 'Installation restrictions' 'OK' 'No device installation restriction policy key found.' 'Low'
}

# 4) Registry-based Explorer / NoDrives / NoViewOnDrive (can hide drive letters)
$explorerPol = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Policies\Explorer'
$noDrives = Get-RegValue $explorerPol 'NoDrives'
$noViewOnDrive = Get-RegValue $explorerPol 'NoViewOnDrive'
if ($null -ne $noDrives -or $null -ne $noViewOnDrive) {
    Add-Report 'Explorer Policy' 'Hidden/restricted drives' 'Review' "NoDrives=$noDrives, NoViewOnDrive=$noViewOnDrive. This can hide or restrict drive access in Explorer." 'Medium'
} else {
    Add-Report 'Explorer Policy' 'Hidden/restricted drives' 'OK' 'No Explorer drive hiding policy found for current user.' 'Low'
}

# 5) Mounted disks and USB disks
$usbDisks = Get-CimInstance Win32_DiskDrive | Where-Object { $_.InterfaceType -eq 'USB' -or $_.PNPDeviceID -match '^USB' }
if ($usbDisks) {
    foreach ($disk in $usbDisks) {
        Add-Report 'Detected Devices' 'USB DiskDrive' 'Found' "Model=$($disk.Model); DeviceID=$($disk.DeviceID); Size=$([math]::Round($disk.Size/1GB,2)) GB; PNPDeviceID=$($disk.PNPDeviceID)" 'Info'
    }
} else {
    Add-Report 'Detected Devices' 'USB DiskDrive' 'Not Found' 'No USB disk drives detected by WMI.' 'Medium'
}

# 6) Logical disks / drive letters
$logical = Get-CimInstance Win32_LogicalDisk | Select-Object DeviceID, VolumeName, DriveType, FileSystem, Size, FreeSpace
if ($logical) {
    foreach ($d in $logical) {
        $sizeGB = if ($d.Size) { [math]::Round($d.Size/1GB,2) } else { $null }
        $freeGB = if ($d.FreeSpace) { [math]::Round($d.FreeSpace/1GB,2) } else { $null }
        Add-Report 'Detected Volumes' 'LogicalDisk' 'Found' "Drive=$($d.DeviceID); Type=$($d.DriveType); FileSystem=$($d.FileSystem); SizeGB=$sizeGB; FreeGB=$freeGB; Label=$($d.VolumeName)" 'Info'
    }
}

# 7) PnP present USB devices
try {
    $pnpUsb = Get-PnpDevice -PresentOnly | Where-Object { $_.InstanceId -match '^USB' -or $_.Class -match 'USB|DiskDrive' }
    if ($pnpUsb) {
        foreach ($dev in $pnpUsb | Select-Object -First 30) {
            Add-Report 'PnP Devices' 'Present USB/PnP device' 'Found' "FriendlyName=$($dev.FriendlyName); Class=$($dev.Class); Status=$($dev.Status); InstanceId=$($dev.InstanceId)" 'Info'
        }
    }
} catch {
    Add-Report 'PnP Devices' 'Present USB/PnP device' 'Skipped' 'Get-PnpDevice not available or failed.' 'Low'
}

# 8) Relevant services
$servicesToCheck = 'USBSTOR','PlugPlay','ShellHWDetection','storsvc'
foreach ($svcName in $servicesToCheck) {
    $svc = Get-Service -Name $svcName -ErrorAction SilentlyContinue
    if ($svc) {
        Add-Report 'Services' $svcName 'Info' "Status=$($svc.Status); StartType may require separate review in Services.msc." 'Info'
    }
}

# 9) Event logs: recent device/install/storage errors (best effort)
try {
    $events = Get-WinEvent -FilterHashtable @{ LogName='System'; StartTime=(Get-Date).AddDays(-7) } -MaxEvents 300 |
        Where-Object {
            $_.ProviderName -match 'Kernel-PnP|UserPnp|PartMgr|disk|Ntfs|storahci|stornvme|USB|usbstor'
        } |
        Select-Object -First 30

    if ($events) {
        foreach ($evt in $events) {
            $msg = ($evt.Message -replace "`r|`n", ' ')
            if ($msg.Length -gt 220) { $msg = $msg.Substring(0,220) + '...' }
            Add-Report 'Event Log' $evt.ProviderName 'Review' "Id=$($evt.Id); Level=$($evt.LevelDisplayName); Time=$($evt.TimeCreated); Message=$msg" 'Medium'
        }
    } else {
        Add-Report 'Event Log' 'Recent relevant system events' 'OK' 'No obvious recent USB/storage-related events found in last 7 days.' 'Low'
    }
} catch {
    Add-Report 'Event Log' 'Recent relevant system events' 'Skipped' 'Unable to read System event log.' 'Low'
}

# 10) Summary heuristics
$blockedItems = $report | Where-Object { $_.Status -eq 'Blocked' }
if ($blockedItems.Count -gt 0) {
    Add-Report 'Summary' 'Likely root cause' 'Attention Needed' ("One or more blocking policies/services were found: " + (($blockedItems | ForEach-Object { $_.Check }) -join '; ')) 'High'
} elseif (($report | Where-Object { $_.Check -eq 'USB DiskDrive' -and $_.Status -eq 'Not Found' }).Count -gt 0) {
    Add-Report 'Summary' 'Likely root cause' 'Review Hardware/Driver' 'No USB disk detected. Possible causes: script blocked driver/service earlier, device driver issue, bad USB port/cable/device, or power issue.' 'Medium'
} else {
    Add-Report 'Summary' 'Likely root cause' 'No direct block found' 'No obvious machine-level USB storage block found. Next checks: specific device, driver cleanup, local user policy, endpoint security, or the original EXE script logic.' 'Medium'
}

# Output files
$ts = Get-Date -Format 'yyyyMMdd_HHmmss'
$outDir = Join-Path $env:USERPROFILE "Desktop\USB_Diagnostic_$ts"
New-Item -ItemType Directory -Path $outDir -Force | Out-Null

$csvPath = Join-Path $outDir 'USB_Block_Report.csv'
$txtPath = Join-Path $outDir 'USB_Block_Report.txt'

$report | Export-Csv -NoTypeInformation -Encoding UTF8 -Path $csvPath
$report | Format-Table -AutoSize | Out-String -Width 500 | Set-Content -Encoding UTF8 -Path $txtPath

Write-Host "`nUSB diagnostic completed." -ForegroundColor Green
Write-Host "Report folder: $outDir" -ForegroundColor Cyan
Write-Host "CSV report   : $csvPath" -ForegroundColor Cyan
Write-Host "TXT report   : $txtPath" -ForegroundColor Cyan
Write-Host "`nTop findings:" -ForegroundColor Yellow
$report | Where-Object { $_.Status -in @('Blocked','Warning','Attention Needed','Review') } | Select-Object -First 15 | Format-Table -AutoSize
