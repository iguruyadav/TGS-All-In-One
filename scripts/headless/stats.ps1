<#
.SYNOPSIS
    TGS Real-time Stats Tool
    Returns CPU, Memory, Disk, and Network usage as JSON
#>

$ErrorActionPreference = "SilentlyContinue"

# 1. CPU Usage
$CPU = (Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average
if ($null -eq $CPU) { $CPU = 0 }

# 2. Memory Usage
$OS = Get-CimInstance Win32_OperatingSystem
$TotalRAM = $OS.TotalVisibleMemorySize
$FreeRAM = $OS.FreePhysicalMemory
$UsedRAM = $TotalRAM - $FreeRAM
$RAMPercent = [math]::Round(($UsedRAM / $TotalRAM) * 100, 1)
$AvailRAMGB = [math]::Round($FreeRAM / 1MB, 2)

# 3. Disk Activity
$Disk = (Get-Counter '\PhysicalDisk(_Total)\% Disk Time' -ErrorAction SilentlyContinue).CounterSamples.CookedValue
if ($Disk -gt 100) { $Disk = 100 }

# 4. Network Usage (Bytes/sec - average over 50ms)
$Net1 = Get-NetAdapterStatistics | Select-Object -Property ReceivedBytes, SentBytes
Start-Sleep -Milliseconds 50
$Net2 = Get-NetAdapterStatistics | Select-Object -Property ReceivedBytes, SentBytes

$In = 0
$Out = 0
for ($i = 0; $i -lt $Net1.Count; $i++) {
    $In += ($Net2[$i].ReceivedBytes - $Net1[$i].ReceivedBytes)
    $Out += ($Net2[$i].SentBytes - $Net1[$i].SentBytes)
}

# Values per second (since we slept 50ms, multiply by 20)
$In = [math]::Round(($In * 20) / 1KB, 2)   # KB/s
$Out = [math]::Round(($Out * 20) / 1KB, 2) # KB/s

# 5. Top 5 Processes by Memory
$TopProcesses = Get-Process | Sort-Object WorkingSet -Descending | Select-Object -First 5 | ForEach-Object {
    [PSCustomObject]@{
        Name = $_.ProcessName
        Mem  = "{0:N0} MB" -f ($_.WorkingSet / 1MB)
    }
}

$Result = [PSCustomObject]@{
    CPU          = [math]::Round($CPU, 1)
    RAM          = @{
        Percent   = $RAMPercent
        Used      = "{0:N2} GB" -f ($UsedRAM / 1MB)
        Total     = "{0:N2} GB" -f ($TotalRAM / 1MB)
        Available = "$AvailRAMGB GB"
    }
    Disk         = [math]::Round($Disk, 1)
    Network      = @{
        In  = $In
        Out = $Out
    }
    TopProcesses = $TopProcesses
}

$Result | ConvertTo-Json -Compress
