# Get real USB device manufacturer names from registry
$results = @()

# USB HID Mice and Keyboards from registry
$usbPath = "HKLM:\SYSTEM\CurrentControlSet\Enum\USB"
if (Test-Path $usbPath) {
    Get-ChildItem $usbPath | ForEach-Object {
        $vid = $_.PSChildName
        Get-ChildItem $_.PSPath | ForEach-Object {
            $subPath = $_.PSPath
            try {
                $props = Get-ItemProperty $subPath -ErrorAction Stop
                $service = $props.Service
                $desc = $props.DeviceDesc
                $mfr = $props.Mfg
                if ($service -like "*mouse*" -or $service -like "*mouhid*" -or $desc -like "*mouse*") {
                    $results += "MOUSE: VID=$vid Desc=$desc Mfg=$mfr"
                }
                if ($service -like "*keyboard*" -or $service -like "*kbdhid*" -or $desc -like "*keyboard*") {
                    $results += "KB: VID=$vid Desc=$desc Mfg=$mfr"
                }
            } catch {}
        }
    }
}

Write-Host "=USB DEVICES="
$results | ForEach-Object { Write-Host $_ }

# HID path for friendly names
Write-Host "=HID="
$hidPath = "HKLM:\SYSTEM\CurrentControlSet\Enum\HID"
if (Test-Path $hidPath) {
    Get-ChildItem $hidPath | ForEach-Object {
        $vidPid = $_.PSChildName
        Get-ChildItem $_.PSPath | ForEach-Object {
            Get-ChildItem $_.PSPath | ForEach-Object {
                try {
                    $props = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
                    if ($props.DeviceDesc -and ($props.Service -like "*mouse*" -or $props.Service -like "*keyboard*")) {
                        Write-Host "HID $vidPid : $($props.DeviceDesc) | $($props.Mfg)"
                    }
                } catch {}
            }
        }
    }
}

# EDID monitor detection
Write-Host "=MONITORS EDID="
$dispPath = "HKLM:\SYSTEM\CurrentControlSet\Enum\DISPLAY"
if (Test-Path $dispPath) {
    Get-ChildItem $dispPath | ForEach-Object {
        $monId = $_.PSChildName  # e.g. DEL4098, ACR001B, GSM5BBF
        Write-Host "Monitor code: $monId"
    }
}

Write-Host "=AUDIO DEVICES="
Get-WmiObject Win32_SoundDevice | Select-Object Name, Manufacturer | Format-List
