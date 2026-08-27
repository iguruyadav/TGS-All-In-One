$out = @{Mice=@(); Keyboards=@(); Monitors=@(); Webcams=@(); Audio=@()}

try {
    $items = Get-WmiObject -Query "SELECT Name,Manufacturer,PNPDeviceID FROM Win32_PnPEntity WHERE PNPClass='Mouse'"
    foreach ($d in $items) {
        $label = "$($d.Manufacturer) $($d.Name)".Trim()
        if ($label -ne "") { $out.Mice += $label }
    }
} catch {}

try {
    $items = Get-WmiObject -Query "SELECT Name,Manufacturer,PNPDeviceID FROM Win32_PnPEntity WHERE PNPClass='Keyboard'"
    foreach ($d in $items) {
        $label = "$($d.Manufacturer) $($d.Name)".Trim()
        if ($label -ne "") { $out.Keyboards += $label }
    }
} catch {}

try {
    $items = Get-WmiObject -Query "SELECT Name,Manufacturer FROM Win32_PnPEntity WHERE PNPClass='Monitor'"
    foreach ($d in $items) {
        $label = "$($d.Manufacturer) $($d.Name)".Trim()
        if ($label -ne "") { $out.Monitors += $label }
    }
} catch {}

try {
    $items = Get-WmiObject -Query "SELECT Name,Manufacturer FROM Win32_PnPEntity WHERE PNPClass='Image'"
    foreach ($d in $items) {
        $label = "$($d.Manufacturer) $($d.Name)".Trim()
        if ($label -ne "") { $out.Webcams += $label }
    }
} catch {}

try {
    $items = Get-WmiObject Win32_SoundDevice
    foreach ($d in $items) {
        $label = "$($d.Manufacturer) $($d.Name)".Trim()
        if ($label -ne "") { $out.Audio += $label }
    }
} catch {}

Write-Host "=MICE="; $out.Mice | ForEach-Object { Write-Host $_ }
Write-Host "=KBD="; $out.Keyboards | ForEach-Object { Write-Host $_ }
Write-Host "=MON="; $out.Monitors | ForEach-Object { Write-Host $_ }
Write-Host "=CAM="; $out.Webcams | ForEach-Object { Write-Host $_ }
Write-Host "=AUD="; $out.Audio | ForEach-Object { Write-Host $_ }
