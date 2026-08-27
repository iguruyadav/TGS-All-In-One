<#
.SYNOPSIS
    TGS Headless Audit Tool
    Returns System Info & Security Status as JSON
#>

$ErrorActionPreference = "SilentlyContinue"

# --- Functions ---
function Get-NetInfo {
    # Exclude virtual, VPN, and Fortinet adapters — only real LAN / Wi-Fi
    $VirtPattern = "Virtual|VMware|VirtualBox|Hyper-V|Fortinet|FortiClient|FortiGate|TAP|Tunnel|VPN|WAN.Mini"
    try {
        $All = Get-NetAdapter -ErrorAction SilentlyContinue | Where-Object { $_.Name -notmatch $VirtPattern }

        # 1. Best default-route adapter that is physical
        $Routes = Get-NetRoute -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue | Sort-Object RouteMetric
        $Selected = $null
        $R = $null
        foreach ($route in $Routes) {
            $Candidate = $All | Where-Object { $_.InterfaceIndex -eq $route.InterfaceIndex } | Select-Object -First 1
            if ($Candidate) { $Selected = $Candidate; $R = $route; break }
        }

        # 2. WMI fallback — skip VPN/Fortinet descriptions
        if (-not $Selected) {
            $WMI = Get-CimInstance Win32_NetworkAdapterConfiguration | Where-Object {
                $_.IPEnabled -and $_.DefaultIPGateway -and $_.Description -notmatch $VirtPattern
            } | Select-Object -First 1
            if ($WMI) {
                $Selected = $All | Where-Object {
                    $_.InterfaceIndex -eq $WMI.InterfaceIndex -or $_.MACAddress -eq $WMI.MACAddress
                } | Select-Object -First 1
                if (-not $Selected) {
                    return @{
                        Name     = $WMI.Description
                        IP       = $WMI.IPAddress[0]
                        MAC      = $WMI.MACAddress
                        Gateway  = $WMI.DefaultIPGateway[0]
                        Internet = $true
                    }
                }
            }
        }

        # 3. Any Up physical adapter
        if (-not $Selected) {
            $Selected = $All | Where-Object { $_.Status -eq "Up" } | Sort-Object Speed -Descending | Select-Object -First 1
        }

        if ($Selected) {
            $IPInfo = Get-NetIPAddress -InterfaceIndex $Selected.InterfaceIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue |
            Where-Object { $_.PrefixOrigin -ne "WellKnown" } | Select-Object -First 1
            if (-not $IPInfo) {
                $IPInfo = Get-NetIPAddress -InterfaceIndex $Selected.InterfaceIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue | Select-Object -First 1
            }
            $AdapterRoute = Get-NetRoute -InterfaceIndex $Selected.InterfaceIndex -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue | Sort-Object RouteMetric | Select-Object -First 1
            $GW = if ($AdapterRoute) { $AdapterRoute.NextHop } elseif ($R) { $R.NextHop } else { "N/A" }
            return @{
                Name     = $Selected.Name
                IP       = if ($IPInfo) { $IPInfo.IPAddress } else { "N/A" }
                MAC      = $Selected.MacAddress
                Gateway  = $GW
                Internet = ($null -ne $AdapterRoute)
            }
        }
    }
    catch {}
    return @{ Name = "N/A"; IP = "N/A"; MAC = "N/A"; Gateway = "N/A"; Internet = $false }
}

function Get-FirewallStatus {
    $fw = Get-NetFirewallProfile -Profile Domain, Public, Private | Where-Object { $_.Enabled -eq $true }
    return ($fw.Count -gt 0)
}

function Get-AVStatus {
    $av = Get-CimInstance -Namespace root/SecurityCenter2 -ClassName AntivirusProduct | Select-Object -ExpandProperty displayName
    if ($av) { return ($av -join ", ") } else { return "None" }
}

function Get-IsAdmin {
    return ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# --- Data Gathering ---
# NOTE: Do NOT use -Select (invalid parameter). Use -Property if you need to limit fields.
$OS = Get-CimInstance -ClassName Win32_OperatingSystem
$CS = Get-CimInstance -ClassName Win32_ComputerSystem
$MB = Get-CimInstance -ClassName Win32_BaseBoard
$CPUFull = (Get-CimInstance -ClassName Win32_Processor | Select-Object -First 1).Name
$CPU = $CPUFull -replace "11th Gen Intel\(R\) Core\(TM\) ", "" -replace "Intel\(R\) Core\(TM\) ", "" -replace "Intel\(R\) ", "" -replace "Core\(TM\) ", ""
$RAMSlots = Get-CimInstance -ClassName Win32_PhysicalMemory

$RAMTotalBytes = ($RAMSlots | Measure-Object -Property Capacity -Sum).Sum
$Speed = if($RAMSlots){($RAMSlots | Select-Object -First 1).Speed}else{0}
$Type = "DDR4"
$TotalCapacity = "{0:N0} GB ($Type @ $Speed MHz)" -f ($RAMTotalBytes / 1GB)
$RAMTotal = "{0:N0} GB" -f ($RAMTotalBytes / 1GB)


# Storage: Map only physical disks with valid media type
$PhysicalDisks = Get-CimInstance -Namespace root/Microsoft/Windows/Storage -ClassName MSFT_PhysicalDisk | Where-Object { $_.BusType -ne "USB" }
$DiskArray = @()
foreach ($pd in $PhysicalDisks) {
    if ($null -ne $pd.MediaType) {
        $TypeText = switch ($pd.MediaType) {
            3 { "HDD" }
            4 { "SSD" }
            5 { "SCM" }
            Default { "Disk" }
        }
        $DiskArray += [PSCustomObject]@{
            Label = "$TypeText $([math]::Round($pd.Size / 1GB)) GB"
            Name  = $pd.FriendlyName
        }
    }
}

# Network
$NetData = Get-NetInfo

# Security
$IsAdmin = Get-IsAdmin
$Firewall = Get-FirewallStatus
$AV = Get-AVStatus
$UAC = (Get-ItemProperty HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System).EnableLUA

# Construct Nested Result Object
$Result = [PSCustomObject]@{
    System   = [PSCustomObject]@{
        Name   = $env:COMPUTERNAME
        User   = $env:USERNAME
        Domain = $env:USERDOMAIN
        OS     = $OS.Caption           # Correct property: Caption (not OsName)
        Build  = $OS.BuildNumber
    }
    Hardware = [PSCustomObject]@{
        Model      = "$($CS.Manufacturer) $($CS.Model)".Trim()
        CPU        = $CPU
        MB         = "$($MB.Manufacturer) $($MB.Product)".Trim()
        RAMTotal   = $TotalCapacity
        RAMSlots   = $RAMSlots.Count
        RAMDetails = $RAMSlots | ForEach-Object { "$([math]::Round($_.Capacity / 1GB)) GB ($Type @ $($_.Speed) MHz)" }
        Storage    = $DiskArray
    }
    Network  = [PSCustomObject]@{
        Adapter  = $NetData.Name
        IP       = $NetData.IP
        MAC      = $NetData.MAC
        Gateway  = $NetData.Gateway
        Internet = $NetData.Internet
    }
    Security = [PSCustomObject]@{
        AV       = $AV
        Firewall = $Firewall
        IsAdmin  = $IsAdmin
        UAC      = if ($UAC -eq 1) { "Enabled" } else { "Disabled" }
    }
}

# Output JSON
$Result | ConvertTo-Json -Depth 5 -Compress
