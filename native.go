package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

type Win32_ComputerSystem struct {
	Manufacturer string
	Model        string
	Name         string
	PCSystemType uint32
}

type Win32_BaseBoard struct {
	Manufacturer string
	Product      string
}

type Win32_BIOS struct {
	Version string
}

type Win32_PhysicalMemory struct {
	Capacity         uint64
	Speed            uint32
	PartNumber       string
	SMBIOSMemoryType uint32
	MemoryType       uint16
}

type Win32_PhysicalMemoryArray struct {
	MemoryDevices uint32
}

type MSFT_PhysicalDisk struct {
	FriendlyName string
	Size         uint64
	MediaType    uint16
	BusType      uint16
}

type Win32_DesktopMonitor struct {
	Name         string
	ScreenWidth  uint32
	ScreenHeight uint32
}

type Win32_Keyboard struct {
	Name        string
	Description string
}

type Win32_PointingDevice struct {
	Name         string
	Description  string
	Manufacturer string
}

type Win32_OperatingSystem struct {
	Caption     string
	BuildNumber string
}

type Win32_NetCfg struct {
	Description      string
	DefaultIPGateway []string
	IPEnabled        bool
}

type Win32_AntivirusProduct struct {
	DisplayName string
}

type AuditSystemInfo struct {
	Name         string `json:"Name"`
	User         string `json:"User"`
	Domain       string `json:"Domain"`
	Manufacturer string `json:"Manufacturer"`
	Model        string `json:"Model"`
	OS           string `json:"OS"`
	Build        string `json:"Build"`
	DeviceType   string `json:"DeviceType"`
}

type AuditHardwareInfo struct {
	CPU        string        `json:"CPU"`
	GPU        string        `json:"GPU"`
	MBBrand    string        `json:"MBBrand"`
	MBModel    string        `json:"MBModel"`
	MB         string        `json:"MB"`
	RAMTotal   string        `json:"RAMTotal"`
	RAMSlots   string        `json:"RAMSlots"`
	RAMDetails []string      `json:"RAMDetails"`
	BIOS       string        `json:"BIOS"`
	Storage    []StorageInfo `json:"Storage"`
}

type AuditNetworkInfo struct {
	Adapter  string `json:"Adapter"`
	IP       string `json:"IP"`
	MAC      string `json:"MAC"`
	Gateway  string `json:"Gateway"`
	Internet bool   `json:"Internet"`
}

type AuditSecurityInfo struct {
	AV             string `json:"AV"`
	Firewall       bool   `json:"Firewall"`
	IsAdmin        bool   `json:"IsAdmin"`
	UAC            string `json:"UAC"`
	USBStatus      string `json:"USBStatus"`
	RDPStatus      string `json:"RDPStatus"`
	BrowserPolicy  string `json:"BrowserPolicy"`
	ScheduleStatus string `json:"ScheduleStatus"`
}

type AuditPeripherals struct {
	Monitors  []string `json:"Monitors"`
	Keyboards []string `json:"Keyboards"`
	Mice      []string `json:"Mice"`
	Webcams   []string `json:"Webcams"`
	Audio     []string `json:"Audio"`
}

type AuditData struct {
	System      AuditSystemInfo   `json:"System"`
	Hardware    AuditHardwareInfo `json:"Hardware"`
	Network     AuditNetworkInfo  `json:"Network"`
	Security    AuditSecurityInfo `json:"Security"`
	Peripherals AuditPeripherals  `json:"Peripherals"`
}

type StorageInfo struct {
	Label string `json:"Label"`
	Type  string `json:"Type"`
	Size  string `json:"Size"`
}

type RAMStatsInfo struct {
	Used      string  `json:"Used"`
	Percent   float64 `json:"Percent"`
	Available string  `json:"Available"`
}

type TopProcessInfo struct {
	Name string `json:"Name"`
	Mem  string `json:"Mem"`
}

type VirtualMemInfo struct {
	Total     string `json:"Total"`
	Used      string `json:"Used"`
	Available string `json:"Available"`
	Percent   int    `json:"Percent"`
}

type NetworkStatsInfo struct {
	In  float64 `json:"In"`
	Out float64 `json:"Out"`
}

// StatsData ...
type StatsData struct {
	CPU          float64          `json:"CPU"`
	RAM          RAMStatsInfo     `json:"RAM"`
	Disk         float64          `json:"Disk"`
	Network      NetworkStatsInfo `json:"Network"`
	TopProcesses []TopProcessInfo `json:"TopProcesses"`
	VirtualMem   VirtualMemInfo   `json:"VirtualMem"`
	Uptime       string           `json:"Uptime"`
	LastBoot     string           `json:"LastBoot"`
}

type HealthLog struct {
	Time    string `json:"Time"`
	Level   string `json:"Level"`
	Source  string `json:"Source"`
	EventID int    `json:"EventID"`
	Message string `json:"Message"`
}

type SystemHealth struct {
	StabilityScore float64     `json:"StabilityScore"`
	LastFailures   []HealthLog `json:"LastFailures"`
	CriticalLogs   []HealthLog `json:"CriticalLogs"`
}

// GetSystemHealth analyzes Windows Event Logs and Reliability History
func (a *App) GetSystemHealth() (SystemHealth, error) {
	var h SystemHealth
	h.StabilityScore = 10.0

	script := `
$err = @()

# Get recent System critical+error events (last 7 days)
try {
	$evts = Get-WinEvent -FilterHashtable @{LogName='System'; Level=1,2; StartTime=(Get-Date).AddDays(-7)} -MaxEvents 20 -ErrorAction SilentlyContinue
} catch { $evts = @() }

# Get reliability records
try {
	$relRecs = Get-CimInstance -ClassName Win32_ReliabilityRecords -ErrorAction SilentlyContinue | 
		Where-Object { $_.TimeGenerated -gt (Get-Date).AddDays(-7) } |
		Sort-Object TimeGenerated -Descending |
		Select-Object -First 10
} catch { $relRecs = @() }

# Stability score
$crashCount = 0
try {
	$crashCount = (Get-WinEvent -FilterHashtable @{LogName='System'; Id=41,1001,6008; StartTime=(Get-Date).AddDays(-7)} -ErrorAction SilentlyContinue | Measure-Object).Count
} catch {}
$score = 10.0 - ($crashCount * 1.5)
if ($score -lt 1.0) { $score = 1.0 }

$criticalLogs = $evts | ForEach-Object {
	$msg = ($_.Message -replace '\r\n', ' ' -replace '\n', ' ').Substring(0, [Math]::Min(200, $_.Message.Length))
	@{
		Time = $_.TimeCreated.ToString("yyyy-MM-dd HH:mm")
		Level = if ($_.Level -eq 1) { "Critical" } else { "Error" }
		Source = $_.ProviderName
		EventID = [int]$_.Id
		Message = $msg
	}
}

$lastFailures = $relRecs | ForEach-Object {
	@{
		Time = $_.TimeGenerated.ToString("yyyy-MM-dd HH:mm")
		Level = "Error"
		Source = $_.SourceName
		EventID = [int]$_.EventIdentifier
		Message = ($_.Message -replace '\r\n', ' ').Substring(0, [Math]::Min(150, $_.Message.Length))
	}
}

@{
	StabilityScore = [double]$score
	LastFailures = @($lastFailures)
	CriticalLogs = @($criticalLogs)
} | ConvertTo-Json -Depth 5
`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err == nil {
		var result struct {
			StabilityScore float64     `json:"StabilityScore"`
			LastFailures   []HealthLog `json:"LastFailures"`
			CriticalLogs   []HealthLog `json:"CriticalLogs"`
		}
		if jsonErr := json.Unmarshal(out, &result); jsonErr == nil {
			h.StabilityScore = result.StabilityScore
			h.LastFailures = enrichLogs(result.LastFailures)
			h.CriticalLogs = enrichLogs(result.CriticalLogs)
		}
	}

	return h, nil
}

// enrichLogs adds troubleshooting tips based on Event ID
func enrichLogs(logs []HealthLog) []HealthLog {
	tips := map[int]string{
		41:    "Fix: Run 'sfc /scannow' and check RAM with 'mdsched.exe'. Check PSU stability.",
		1001:  "BugCheck BSOD. Run 'chkdsk /f' and 'winmemdiag'. Check minidumps in C:\\Windows\\Minidump.",
		6008:  "Unexpected shutdown. Check UPS/power supply. Run Event Viewer > System for patterns.",
		7034:  "Service crashed. Check Services.msc and Event Viewer for the failing service name.",
		7031:  "Service failed unexpectedly. Review Application log for crash reason.",
		7023:  "Service terminated with error. Check dependent services in Services.msc.",
		7001:  "Service logon failure. Check service account credentials in Services.msc.",
		55:    "Disk corruption detected. Run 'chkdsk C: /f /r' from Admin CMD.",
		15:    "Disk I/O error. Check SMART status with CrystalDiskInfo. Replace failing disk.",
		157:   "Disk surprise removed. Check SATA/USB cable and power connectors.",
		153:   "Disk bad block. Run 'chkdsk /r'. Consider replacing the drive soon.",
		129:   "Disk reset. Could be overheating or failing HDD/SSD. Check temps.",
		1000:  "Application crash. Check Application log for the process name and fault module.",
		1002:  "Application hang. Close hanging process or reinstall the application.",
		10010: "DCOM server unavailable. Run 'dcomcnfg' or 'sfc /scannow'.",
		10016: "DCOM permission error. May be benign; registry permission issue in DCOM.",
		51:    "Page fault during disk I/O. Run 'chkdsk' and check RAM.",
		4:     "Driver error. Update or roll back the device driver listed in the error.",
		219:   "Driver failed to load. Update or reinstall the relevant driver.",
		5973:  "Personalization API failure. May indicate profile corruption. Create new user profile.",
		46:    "Crash dump failed — ensure adequate disk space on system drive.",
	}
	for i := range logs {
		if tip, ok := tips[logs[i].EventID]; ok {
			logs[i].Message = logs[i].Message + " | 💡 " + tip
		}
	}
	return logs
}

func (a *App) GetNativeAudit() (AuditData, error) {
	var data AuditData

	// System Info
	var cs []Win32_ComputerSystem
	if err := wmi.Query("SELECT Manufacturer, Model, Name FROM Win32_ComputerSystem", &cs); err == nil && len(cs) > 0 {
		data.System.Manufacturer = cs[0].Manufacturer
		data.System.Model = cs[0].Model
		data.System.Name = cs[0].Name
		// If WMI returned empty name, fall back to os.Hostname
		if data.System.Name == "" {
			data.System.Name, _ = os.Hostname()
		}
		// If model is empty or generic, try from BaseBoard
		genericModels := []string{"System Product Name", "To be filled by O.E.M.", "Default string", "--"}
		isGeneric := false
		for _, g := range genericModels {
			if strings.EqualFold(data.System.Model, g) || data.System.Model == "" {
				isGeneric = true
				break
			}
		}
		if isGeneric {
			var bb []Win32_BaseBoard
			if err := wmi.Query("SELECT Product FROM Win32_BaseBoard", &bb); err == nil && len(bb) > 0 {
				data.System.Model = bb[0].Product
			} else {
				data.System.Model = cs[0].Manufacturer
			}
		}
		// PCSystemType: 2 = Mobile (Laptop), all other values = Desktop
		if cs[0].PCSystemType == 2 {
			data.System.DeviceType = "Laptop"
		} else {
			data.System.DeviceType = "Desktop PC"
		}
	} else {
		// WMI query failed — use hostname
		data.System.Name, _ = os.Hostname()
	}

	// OS Info
	var osInfo []Win32_OperatingSystem
	if err := wmi.Query("SELECT Caption, BuildNumber FROM Win32_OperatingSystem", &osInfo); err == nil && len(osInfo) > 0 {
		data.System.OS = osInfo[0].Caption
		data.System.Build = osInfo[0].BuildNumber
	}

	// Current user & domain from environment
	data.System.User = os.Getenv("USERNAME")
	data.System.Domain = os.Getenv("USERDOMAIN")

	var bb []Win32_BaseBoard
	if err := wmi.Query("SELECT Manufacturer, Product FROM Win32_BaseBoard", &bb); err == nil && len(bb) > 0 {
		data.Hardware.MBBrand = strings.TrimSpace(bb[0].Manufacturer)
		data.Hardware.MBModel = strings.TrimSpace(bb[0].Product)
		data.Hardware.MB = fmt.Sprintf("%s %s", data.Hardware.MBBrand, data.Hardware.MBModel)
	}

	var bios []Win32_BIOS
	if err := wmi.Query("SELECT Version FROM Win32_BIOS", &bios); err == nil && len(bios) > 0 {
		data.Hardware.BIOS = bios[0].Version
	}

	cpus, _ := cpu.Info()
	if len(cpus) > 0 {
		name := cpus[0].ModelName
		// Strip Intel/Core branding
		name = strings.ReplaceAll(name, "11th Gen ", "")
		name = strings.ReplaceAll(name, "Intel(R) Core(TM) ", "")
		name = strings.ReplaceAll(name, "Intel(R) ", "")
		name = strings.ReplaceAll(name, "Core(TM) ", "")
		data.Hardware.CPU = strings.TrimSpace(name)
	}

	// GPU Info via WMI
	type Win32_VideoCtrl struct {
		Name                 string
		AdapterRAM           uint32
		CurrentRefreshRate   uint32
		VideoModeDescription string
	}
	var gpus []Win32_VideoCtrl
	if err := wmi.Query("SELECT Name, AdapterRAM FROM Win32_VideoController", &gpus); err == nil && len(gpus) > 0 {
		var gpuParts []string
		for _, g := range gpus {
			ramMB := g.AdapterRAM / 1024 / 1024
			if g.Name != "" && !strings.Contains(strings.ToLower(g.Name), "remote") {
				if ramMB > 0 {
					gpuParts = append(gpuParts, fmt.Sprintf("%s (%d MB)", g.Name, ramMB))
				} else {
					gpuParts = append(gpuParts, g.Name)
				}
			}
		}
		if len(gpuParts) > 0 {
			data.Hardware.GPU = strings.Join(gpuParts, " | ")
		}
	}

	var memory []Win32_PhysicalMemory
	if err := wmi.Query("SELECT Capacity, Speed, PartNumber, SMBIOSMemoryType, MemoryType FROM Win32_PhysicalMemory", &memory); err == nil {
		var totalCapacity uint64
		var firstSpeed uint32
		var ddrType string = "DDR4" // Default

		for _, m := range memory {
			totalCapacity += m.Capacity
			if firstSpeed == 0 {
				firstSpeed = m.Speed
			}
			
			smType := m.SMBIOSMemoryType
			if smType == 0 {
				smType = uint32(m.MemoryType)
			}

			if smType == 34 {
				ddrType = "DDR5"
			} else if smType == 26 {
				ddrType = "DDR4"
			} else if smType == 24 {
				ddrType = "DDR3"
			} else if smType == 21 {
				ddrType = "DDR2"
			} else if smType == 20 {
				ddrType = "DDR"
			} else if smType == 0 {
				if m.Speed >= 4800 {
					ddrType = "DDR5"
				} else if m.Speed > 2666 {
					ddrType = "DDR4"
				} else if m.Speed < 2133 && m.Speed > 1066 {
					ddrType = "DDR3"
				}
			}

			data.Hardware.RAMDetails = append(data.Hardware.RAMDetails, fmt.Sprintf("%d GB (%s @ %d MHz)", m.Capacity/1024/1024/1024, ddrType, m.Speed))
		}
		
		data.Hardware.RAMTotal = fmt.Sprintf("%d GB (%s @ %d MHz)", totalCapacity/1024/1024/1024, ddrType, firstSpeed)
		
		var memArray []Win32_PhysicalMemoryArray
		if errArray := wmi.Query("SELECT MemoryDevices FROM Win32_PhysicalMemoryArray", &memArray); errArray == nil && len(memArray) > 0 {
			data.Hardware.RAMSlots = fmt.Sprintf("%d of %d", len(memory), memArray[0].MemoryDevices)
		} else {
			data.Hardware.RAMSlots = fmt.Sprintf("%d", len(memory))
		}
	}

	// Storage Info
	var disks []MSFT_PhysicalDisk
	if err := wmi.QueryNamespace("SELECT FriendlyName, Size, MediaType, BusType FROM MSFT_PhysicalDisk", &disks, "root\\Microsoft\\Windows\\Storage"); err == nil {
		for _, d := range disks {
			if d.BusType == 7 {
				continue
			}
			media := "HDD"
			if d.MediaType == 4 {
				media = "SSD"
			}
			data.Hardware.Storage = append(data.Hardware.Storage, StorageInfo{
				Label: d.FriendlyName,
				Type:  media,
				Size:  fmt.Sprintf("%d GB", d.Size/1024/1024/1024),
			})
		}
	}

	// Network Info — skip VPN, virtual, Fortinet adapters
	data.Network.Adapter = "N/A"
	data.Network.Internet = false
	skipPatterns := []string{"virtual", "vmware", "vpn", "hyper-v", "fortinet", "forticlient", "fortigate", "tap", "tunnel", "wan miniport"}
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		isUp := false
		for _, f := range iface.Flags {
			if strings.ToLower(f) == "up" {
				isUp = true
				break
			}
		}
		if !isUp || iface.HardwareAddr == "" {
			continue
		}
		ln := strings.ToLower(iface.Name)
		skip := false
		for _, pat := range skipPatterns {
			if strings.Contains(ln, pat) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		// Accept Ethernet or Wi-Fi
		if strings.Contains(ln, "ethernet") || strings.Contains(ln, "wi-fi") || strings.Contains(ln, "wifi") || strings.Contains(ln, "local area") {
			data.Network.Adapter = iface.Name
			data.Network.MAC = iface.HardwareAddr
			for _, addr := range iface.Addrs {
				if strings.Contains(addr.Addr, ".") && !strings.HasPrefix(addr.Addr, "169.") {
					data.Network.IP = strings.Split(addr.Addr, "/")[0]
					break
				}
			}
			data.Network.Internet = data.Network.IP != ""
			break
		}
	}

	var cfgs []Win32_NetCfg
	if err := wmi.Query("SELECT Description, DefaultIPGateway, IPEnabled FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled=True", &cfgs); err == nil {
		adLower := strings.ToLower(data.Network.Adapter)
		for _, c := range cfgs {
			descLower := strings.ToLower(c.Description)
			isMatch := adLower != "" && (strings.Contains(descLower, adLower) || strings.Contains(adLower, descLower))
			if !isMatch {
				// Skip virtual/VPN adapters in config too
				skip := false
				for _, pat := range skipPatterns {
					if strings.Contains(descLower, pat) {
						skip = true
						break
					}
				}
				if skip {
					continue
				}
			}
			if len(c.DefaultIPGateway) > 0 && c.DefaultIPGateway[0] != "" {
				data.Network.Gateway = c.DefaultIPGateway[0]
				// If we never found adapter earlier, use this config's adapter IP info
				if data.Network.IP == "" && len(cfgs) > 0 {
					data.Network.Adapter = c.Description
				}
				break
			}
		}
	}

	// Security Info
	// AV — via WMI SecurityCenter2
	var avList []Win32_AntivirusProduct
	if err := wmi.QueryNamespace("SELECT DisplayName FROM AntivirusProduct", &avList, "root\\SecurityCenter2"); err == nil && len(avList) > 0 {
		var names []string
		for _, av := range avList {
			names = append(names, av.DisplayName)
		}
		data.Security.AV = strings.Join(names, ", ")
	} else {
		data.Security.AV = "None"
	}
	// Firewall — profile enabled registry key
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile`, registry.QUERY_VALUE)
	if err == nil {
		val, _, _ := k.GetIntegerValue("EnableFirewall")
		data.Security.Firewall = val == 1
		k.Close()
	}
	// UAC
	k2, err2 := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, registry.QUERY_VALUE)
	if err2 == nil {
		uac, _, _ := k2.GetIntegerValue("EnableLUA")
		if uac == 1 {
			data.Security.UAC = "Enabled"
		} else {
			data.Security.UAC = "Disabled"
		}
		k2.Close()
	}
	// IsAdmin — check by attempting to open a privileged registry key
	k3, err3 := registry.OpenKey(registry.LOCAL_MACHINE, `SAM`, registry.QUERY_VALUE)
	if err3 == nil {
		data.Security.IsAdmin = true
		k3.Close()
	}

	// USB Status — read real registry (USBSTOR service Start value + GPO)
	// Start=4 means disabled; Deny_All=1 means blocked.
	data.Security.USBStatus = "ALLOWED"
	usbKey, usbErr := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\USBSTOR`, registry.QUERY_VALUE)
	if usbErr == nil {
		usbStart, _, _ := usbKey.GetIntegerValue("Start")
		if usbStart == 4 {
			data.Security.USBStatus = "BLOCKED"
		}
		usbKey.Close()
	}
	
	// Also check GPO
	gpoUsb, gpoErr := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices`, registry.QUERY_VALUE)
	if gpoErr == nil {
		denyAll, _, _ := gpoUsb.GetIntegerValue("Deny_All")
		if denyAll == 1 {
			data.Security.USBStatus = "BLOCKED"
		}
		gpoUsb.Close()
	}


	// RDP Clipboard Status
	k5, err5 := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, registry.QUERY_VALUE)
	data.Security.RDPStatus = "ALLOWED"
	if err5 == nil {
		val, _, _ := k5.GetIntegerValue("fDisableClip")
		if val == 1 {
			data.Security.RDPStatus = "BLOCKED"
		}
		k5.Close()
	}

	// Browser Policy Status
	k6, err6 := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Google\Chrome`, registry.QUERY_VALUE)
	data.Security.BrowserPolicy = "ALLOWED"
	if err6 == nil {
		_, _, errVal := k6.GetStringValue("AllowedDomainsForApps")
		if errVal == nil {
			data.Security.BrowserPolicy = "BLOCKED"
		}
		k6.Close()
	}

	// Schedule Status
	data.Security.ScheduleStatus = "INACTIVE"
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "if(Get-ScheduledTask -TaskName 'TGS_Weekly_Audit' -ErrorAction SilentlyContinue){'ACTIVE'}else{'INACTIVE'}")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err == nil {
		if strings.Contains(string(out), "ACTIVE") {
			data.Security.ScheduleStatus = "ACTIVE"
		}
	}

	// User requested to remove peripheral auto-detection completely.
	// We leave the fields empty so the Svelte form stays blank.
	data.Peripherals.Monitors = []string{}
	data.Peripherals.Keyboards = []string{}
	data.Peripherals.Mice = []string{}
	data.Peripherals.Webcams = []string{}
	data.Peripherals.Audio = []string{}

	return data, nil
}

func (a *App) GetNativeStats() (StatsData, error) {
	var data StatsData

	// CPU
	c, _ := cpu.Percent(0, false)
	if len(c) > 0 {
		data.CPU = math.Round(c[0]*10) / 10
	}

	// RAM
	v, _ := mem.VirtualMemory()
	data.RAM.Percent = math.Round(v.UsedPercent*10) / 10
	data.RAM.Used = fmt.Sprintf("%.1f GB", float64(v.Used)/1024/1024/1024)
	data.RAM.Available = fmt.Sprintf("%.1f GB", float64(v.Available)/1024/1024/1024)

	// Disk
	d, _ := disk.Usage("C:")
	data.Disk = math.Round(d.UsedPercent*10) / 10

	// Network - Simple delta tracking would be better but for now let's use recent counters
	counters, _ := net.IOCounters(false)
	if len(counters) > 0 {
		data.Network.In = math.Round(float64(counters[0].BytesRecv)/1024/1024*10) / 10
		data.Network.Out = math.Round(float64(counters[0].BytesSent)/1024/1024*10) / 10
	}

	// Processes
	procs, _ := process.Processes()
	type procMem struct {
		Name   string
		Memory float64
	}
	var procList []procMem
	for _, p := range procs {
		n, _ := p.Name()
		if n == "" || n == "Idle" || n == "System" {
			continue
		}
		m, _ := p.MemoryInfo()
		if m != nil {
			procList = append(procList, procMem{Name: n, Memory: float64(m.RSS) / 1024 / 1024})
		}
	}
	sort.Slice(procList, func(i, j int) bool { return procList[i].Memory > procList[j].Memory })
	limit := 5
	if len(procList) < 5 {
		limit = len(procList)
	}
	for i := 0; i < limit; i++ {
		data.TopProcesses = append(data.TopProcesses, struct {
			Name string `json:"Name"`
			Mem  string `json:"Mem"`
		}{
			Name: procList[i].Name,
			Mem:  fmt.Sprintf("%d MB", int(procList[i].Memory)),
		})
	}

	// Virtual Memory (Page File) - Use WMI for real PageFile data
	type PageFileUsage struct {
		AllocatedBaseSize uint32
		CurrentUsage      uint32
		PeakUsage         uint32
	}
	var pfUsage []PageFileUsage
	wmiErr := wmi.Query("SELECT AllocatedBaseSize, CurrentUsage, PeakUsage FROM Win32_PageFileUsage", &pfUsage)
	if wmiErr == nil && len(pfUsage) > 0 {
		totalMB := float64(pfUsage[0].AllocatedBaseSize)
		usedMB := float64(pfUsage[0].CurrentUsage)
		freeMB := totalMB - usedMB
		pct := 0.0
		if totalMB > 0 {
			pct = (usedMB / totalMB) * 100
		}
		data.VirtualMem.Total = fmt.Sprintf("%.1f GB", totalMB/1024)
		data.VirtualMem.Used = fmt.Sprintf("%.1f GB", usedMB/1024)
		data.VirtualMem.Available = fmt.Sprintf("%.1f GB", freeMB/1024)
		data.VirtualMem.Percent = int(pct)
	} else {
		// Fallback to gopsutil SwapMemory
		ms, _ := mem.SwapMemory()
		if ms != nil {
			data.VirtualMem.Total = fmt.Sprintf("%.1f GB", float64(ms.Total)/1024/1024/1024)
			data.VirtualMem.Used = fmt.Sprintf("%.1f GB", float64(ms.Used)/1024/1024/1024)
			data.VirtualMem.Available = fmt.Sprintf("%.1f GB", float64(ms.Free)/1024/1024/1024)
			data.VirtualMem.Percent = int(ms.UsedPercent)
		}
	}

	// Uptime & Last Boot
	if uptimeSec, err := host.Uptime(); err == nil {
		days := uptimeSec / 86400
		hours := (uptimeSec % 86400) / 3600
		mins := (uptimeSec % 3600) / 60
		if days > 0 {
			data.Uptime = fmt.Sprintf("%dd %dh %dm", days, hours, mins)
		} else if hours > 0 {
			data.Uptime = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			data.Uptime = fmt.Sprintf("%dm", mins)
		}
	}
	if bootTime, err := host.BootTime(); err == nil {
		data.LastBoot = time.Unix(int64(bootTime), 0).Format("2006-01-02 15:04")
	}

	return data, nil
}


func (a *App) ApplyNativeAction(action string, args []string) (string, error) {
	hide := &syscall.SysProcAttr{HideWindow: true}
	runPS := func(script string) (string, error) {
		cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-Command", script)
		cmd.SysProcAttr = hide
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	switch action {
	// ── Original action names (legacy) ───────────────────────────
	case "-SetName":
		if len(args) > 0 {
			runPS(fmt.Sprintf("Rename-Computer -NewName '%s' -Force", args[0]))
			return "Name changed. Reboot required.", nil
		}
	case "-EnableRDP":
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Terminal Server`, registry.SET_VALUE)
		defer k.Close()
		k.SetDWordValue("fDenyTSConnections", 0)
		exec.Command("powershell", "-Command", "Enable-NetFirewallRule -DisplayGroup 'Remote Desktop'").Run()
		return "RDP Enabled", nil
	case "-DisableRDP":
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Terminal Server`, registry.SET_VALUE)
		defer k.Close()
		k.SetDWordValue("fDenyTSConnections", 1)
		return "RDP Disabled", nil
	case "-Map121":
		exec.Command("net", "use", "Z:", `\\192.168.1.121\Public`, "/persistent:yes").Run()
		return "Drive Z: mapped", nil
	case "-Remove121":
		exec.Command("net", "use", "Z:", "/delete", "/y").Run()
		return "Drive Z: removed", nil
	case "-Reboot":
		exec.Command("shutdown", "/r", "/t", "0").Run()
		return "Rebooting...", nil

	// ── New Setup Tab action names ────────────────────────────────
	case "set-pc-name":
		if len(args) > 0 && args[0] != "" {
			return runPS(fmt.Sprintf("Rename-Computer -NewName '%s' -Force; Write-Output 'Done'", args[0]))
		}
		return "No name provided", nil

	case "set-timezone":
		return runPS(`
			Start-Service -Name W32Time -ErrorAction SilentlyContinue 2>&1 | Out-Null
			Set-Service -Name W32Time -StartupType Automatic -ErrorAction SilentlyContinue 2>&1 | Out-Null
			tzutil /s "India Standard Time" 2>&1 | Out-Null
			w32tm /resync 2>&1 | Out-Null
			Write-Output "IST set"
		`)
	case "enable-rdp":
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Terminal Server`, registry.SET_VALUE)
		if k != 0 {
			k.SetDWordValue("fDenyTSConnections", 0)
			k.Close()
		}
		exec.Command("powershell", "-Command", "Enable-NetFirewallRule -DisplayGroup 'Remote Desktop'").Run()
		return "RDP Enabled", nil

	case "disable-rdp":
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Terminal Server`, registry.SET_VALUE)
		if k != 0 {
			k.SetDWordValue("fDenyTSConnections", 1)
			k.Close()
		}
		return "RDP Disabled", nil

	case "add-this-pc-icon":
		return runPS(`$p="HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\HideDesktopIcons\NewStartPanel";if(!(Test-Path $p)){New-Item -Path $p -Force|Out-Null};Set-ItemProperty -Path $p -Name "{20D04FE0-3AEA-1069-A2D8-08002B30309D}" -Value 0 -Type DWord -Force;Write-Output "This PC icon added"`)

	case "set-wallpaper":
		// Look for bg.jpg relative to the executable
		exePath, _ := os.Executable()
		exeDir := strings.ReplaceAll(exePath, "\\", "/")
		_ = exeDir
		wallScript := `
$paths = @("$PSScriptRoot\frontend\src\assets\images\bg.jpg","$PSScriptRoot\assets\images\bg.jpg",".\frontend\src\assets\images\bg.jpg")
$wall = $paths | Where-Object { Test-Path $_ } | Select-Object -First 1
if($wall) {
  Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class WP{[DllImport("user32.dll")]public static extern int SystemParametersInfo(int a,int b,string c,int d);}' -ErrorAction SilentlyContinue
  [WP]::SystemParametersInfo(0x0014, 0, (Resolve-Path $wall).Path, 0x1 -bor 0x2)|Out-Null
  Write-Output "Wallpaper set"
} else { Write-Output "bg.jpg not found" }`
		return runPS(wallScript)

	case "add-creds-z":
		return runPS(`
			$ip = '174.156.5.121'
			$user = '174.156.5.121\tgsuser121'
			$pass = '@121#Sgt'
			$out1 = cmdkey /add:$ip /user:$user /pass:$pass 2>&1
			Write-Output "Add Creds Result: $out1"
		`)

	case "map-z-drive":
		return runPS(`
			net use Z: /delete /y 2>&1 | Out-Null
			net use X: /delete /y 2>&1 | Out-Null
			net use V: /delete /y 2>&1 | Out-Null
			
			$out1 = net use Z: \\174.156.5.121\Software /user:174.156.5.121\tgsuser121 '@121#Sgt' /persistent:yes 2>&1
			$out2 = net use X: \\174.156.5.121\Extra /user:174.156.5.121\tgsuser121 '@121#Sgt' /persistent:yes 2>&1
			$out3 = net use V: "\\174.156.5.121\Designer Backup" /user:174.156.5.121\tgsuser121 '@121#Sgt' /persistent:yes 2>&1
			
			if ($LASTEXITCODE -eq 0) {
				Write-Output "Map Z,X,V Result: Success"
			} else {
				Write-Output "Map Z,X,V Result: Z($out1) X($out2) V($out3)"
			}
		`)

	case "unmap-z-drive":
		return runPS(`
			net use Z: /delete /y 2>&1 | Out-Null
			net use X: /delete /y 2>&1 | Out-Null
			net use V: /delete /y 2>&1 | Out-Null
			
			$out2 = cmdkey /delete:174.156.5.121 2>&1
			Write-Output "Unmap Server 121 Drives Result: Success | $out2"
		`)

	case "add-creds-y":
		if len(args) >= 2 {
			u := strings.ReplaceAll(args[0], "'", "")
			p := strings.ReplaceAll(args[1], "'", "")
			return runPS(fmt.Sprintf(`$out1 = cmdkey /add:174.156.4.3 /user:%s /pass:%s 2>&1; Write-Output "Add Creds NAS Result: $out1"`, u, p))
		}
		return "Credentials required", nil

	case "map-y-drive":
		return runPS(`net use Y: /delete /y 2>&1 | Out-Null; $out2 = net use Y: \\174.156.4.3\NAS /persistent:yes 2>&1; Write-Output "Map Y Result: $out2"`)

	case "unmap-y-drive":
		return runPS(`$out1 = net use Y: /delete /y 2>&1; $out2 = cmdkey /delete:174.156.4.3 2>&1; Write-Output "Unmap Y Result: $out1 | $out2"`)

	case "upgrade-windows":
		// Try changepk.exe first (works on most systems without reboot)
		// Fall back to slmgr approach if it fails
		ps := `
$key = "VK7JG-NPHTM-C97JM-9MPGT-3V66T"
# Method 1: changepk.exe
$cpk = "$env:SystemRoot\System32\changepk.exe"
if (Test-Path $cpk) {
    Write-Output "Trying changepk.exe..."
    Start-Process -FilePath $cpk -ArgumentList "/ProductKey $key" -Wait -PassThru | Out-Null
    Write-Output "Upgrade initiated via changepk.exe"
} else {
    # Method 2: slmgr.vbs
    Write-Output "Trying slmgr.vbs..."
    cscript //B C:\Windows\System32\slmgr.vbs /ipk $key
    Start-Sleep 2
    cscript //B C:\Windows\System32\slmgr.vbs /ato
    Write-Output "Key applied via slmgr.vbs"
}
`
		return runPS(ps)

	case "activate-windows":
		// Temporarily disable Defender real-time protection, run MAS, then re-enable
		// The inner PS command is passed as a variable to avoid backtick conflicts
		masCmd := "try { irm https://get.activated.win | iex } catch { Write-Host 'Download failed. Check internet.' -ForegroundColor Red }"
		ps := fmt.Sprintf(`
$ErrorActionPreference = "SilentlyContinue"
Write-Output "Disabling Defender real-time protection temporarily..."
Set-MpPreference -DisableRealtimeMonitoring $true 2>$null
Start-Sleep 1
Write-Output "Launching Microsoft Activation Scripts (MAS)..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "%s"
Start-Sleep 5
Write-Output "Re-enabling Defender real-time protection..."
Set-MpPreference -DisableRealtimeMonitoring $false 2>$null
Write-Output "Activation window opened. Defender re-enabled."
`, masCmd)
		return runPS(ps)

	// ── Scheduled Task ─────────────────────────────────────────────
	case "create-audit-schedule":
		exePath, _ := os.Executable()
		ps := fmt.Sprintf(`
$action  = New-ScheduledTaskAction -Execute '%s' -Argument '-silent'
$trigger = @(
    $(New-ScheduledTaskTrigger -Weekly -WeeksInterval 1 -DaysOfWeek Monday -At 12:00PM),
    $(New-ScheduledTaskTrigger -Weekly -WeeksInterval 1 -DaysOfWeek Friday -At 12:00PM)
)
$settings = New-ScheduledTaskSettingsSet -RunOnlyIfNetworkAvailable -StartWhenAvailable
Register-ScheduledTask -TaskName 'TGS_Weekly_Audit' -Action $action -Trigger $trigger -Settings $settings -RunLevel Highest -Force | Out-Null
Write-Output "SCHEDULED"
`, exePath)
		out, err := runPS(ps)
		if err != nil || !strings.Contains(out, "SCHEDULED") {
			return "", fmt.Errorf("schedule creation failed: %v %s", err, out)
		}
		return "Schedule created — Mon & Fri at 12:00 PM", nil

	case "delete-audit-schedule":
		out, err := runPS(`Unregister-ScheduledTask -TaskName 'TGS_Weekly_Audit' -Confirm:$false -ErrorAction SilentlyContinue; Write-Output "DONE"`)
		if err != nil {
			return "", fmt.Errorf("delete failed: %v %s", err, out)
		}
		return "Schedule removed", nil

	case "reboot":
		exec.Command("shutdown", "/r", "/t", "0").Run()
		return "Rebooting...", nil

	case "quick-scan":
		// Trigger Defender quick scan in background
		cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-NonInteractive", "-Command",
			`Start-MpScan -ScanType QuickScan -ErrorAction Stop; Write-Output "Quick Scan started in background"`)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("Quick Scan failed: %s", strings.TrimSpace(string(out)))
		}
		return strings.TrimSpace(string(out)), nil

	case "restart-service":
		if len(args) > 0 {
			svc := strings.ReplaceAll(args[0], "'", "")
			out, err := runPS(fmt.Sprintf(`
try {
    Restart-Service -Name '%s' -Force -ErrorAction Stop
    Write-Output "Service '%s' restarted successfully."
} catch {
    Write-Output "Error: $($_.Exception.Message)"
}
`, svc, svc))
			if err != nil {
				return "Failed: " + out, nil
			}
			return out, nil
		}
		return "No service name provided", nil

	case "open-win-security":
		// Open Windows Security app
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			`Start-Process "windowsdefender:" -ErrorAction SilentlyContinue; Write-Output "Windows Security opened"`)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Start()
		return "Windows Security opened", nil

	case "open-firewall-rules":
		// Open Windows Defender Firewall with Advanced Security (inbound/outbound rules)
		cmd := exec.Command("mmc.exe", "wf.msc")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
		cmd.Start()
		return "Firewall Rules page opened (wf.msc)", nil

	// ── Security actions ──────────────────────────────────────────
	case "block-usb":
		// 1. Disable USBSTOR service
		usbK, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\USBSTOR`, registry.SET_VALUE)
		if usbK != 0 {
			usbK.SetDWordValue("Start", 4)
			usbK.Close()
		}

		// 2. Enable Deny_All in Group Policy
		gpoPath := `SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices`
		gpoK, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, gpoPath, registry.SET_VALUE)
		if gpoK != 0 {
			gpoK.SetDWordValue("Deny_All", 1)
			gpoK.Close()
		}

		// 3. Restart Explorer to apply immediately
		exec.Command("powershell", "-Command", "Get-Process explorer -ErrorAction SilentlyContinue | Stop-Process -Force; Start-Process explorer.exe").Start()

		return "USB storage BLOCKED (Service & GPO). Applied immediately.", nil

	case "allow-usb":
		// 1. Re-enable USBSTOR service
		usbK, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\USBSTOR`, registry.SET_VALUE)
		if usbK != 0 {
			usbK.SetDWordValue("Start", 3)
			usbK.Close()
		}

		// 2. Remove Deny_All in Group Policy
		gpoPath := `SOFTWARE\Policies\Microsoft\Windows\RemovableStorageDevices`
		gpoK, _ := registry.OpenKey(registry.LOCAL_MACHINE, gpoPath, registry.SET_VALUE)
		if gpoK != 0 {
			gpoK.DeleteValue("Deny_All")
			gpoK.DeleteValue("Deny_Read")
			gpoK.DeleteValue("Deny_Write")
			gpoK.Close()
		}

		gpoUserK, _ := registry.OpenKey(registry.CURRENT_USER, gpoPath, registry.SET_VALUE)
		if gpoUserK != 0 {
			gpoUserK.DeleteValue("Deny_All")
			gpoUserK.DeleteValue("Deny_Read")
			gpoUserK.DeleteValue("Deny_Write")
			gpoUserK.Close()
		}

		// 3. Restart Explorer to apply immediately
		exec.Command("powershell", "-Command", "Get-Process explorer -ErrorAction SilentlyContinue | Stop-Process -Force; Start-Process explorer.exe").Start()

		return "USB storage ALLOWED. Reconnect USB drives if needed.", nil

	case "block-rdp-clip":
		p := `SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, p, registry.SET_VALUE)
		if err != nil {
			registry.CreateKey(registry.LOCAL_MACHINE, p, registry.SET_VALUE)
			k, err = registry.OpenKey(registry.LOCAL_MACHINE, p, registry.SET_VALUE)
			if err != nil {
				return "", fmt.Errorf("cannot open RDP Terminal Services key: %v", err)
			}
		}
		k.SetDWordValue("fDisableClip", 1)
		k.Close()
		return "RDP Clipboard BLOCKED", nil

	case "allow-rdp-clip":
		p := `SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, p, registry.SET_VALUE)
		if k != 0 {
			k.DeleteValue("fDisableClip")
			k.Close()
		}
		return "RDP Clipboard ALLOWED", nil

	case "apply-browser-policy":
		domains := ""
		if len(args) > 0 {
			domains = args[0]
		}
		if domains == "" {
			return "No domains provided", nil
		}

		domainList := strings.Split(domains, ",")
		chromKeys := []string{
			`SOFTWARE\Policies\Google\Chrome`,
			`SOFTWARE\Policies\Microsoft\Edge`,
			`SOFTWARE\Policies\BraveSoftware\Brave`,
		}
		ffPath := `SOFTWARE\Policies\Mozilla\Firefox`
		ffFilter := ffPath + `\WebsiteFilter`

		webDomains := []string{}
		ffExceptions := []string{}
		profileRegex := []string{}

		for _, d := range domainList {
			clean := strings.TrimSpace(strings.ReplaceAll(d, "*@", ""))
			if clean != "" {
				webDomains = append(webDomains, clean)
				ffExceptions = append(ffExceptions, "https://"+clean, "https://www."+clean)

				if strings.Contains(clean, "gmail.com") || strings.Contains(clean, "google.com") {
					foundAccounts := false
					for _, wd := range webDomains {
						if wd == "accounts.google.com" {
							foundAccounts = true
							break
						}
					}
					if !foundAccounts {
						webDomains = append(webDomains, "accounts.google.com")
					}
					ffExceptions = append(ffExceptions, "https://accounts.google.com", "https://mail.google.com")
				}

				reg := strings.ReplaceAll(strings.TrimSpace(d), ".", `\.`)
				reg = strings.ReplaceAll(reg, "*", ".*")
				profileRegex = append(profileRegex, reg)
			}
		}

		webString := strings.Join(webDomains, ",")
		profileString := strings.Join(profileRegex, "|")

		for _, kPath := range chromKeys {
			k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, kPath, registry.ALL_ACCESS)
			if k != 0 {
				k.DeleteValue("IncognitoModeAvailability")
				k.DeleteValue("BrowserGuestModeEnabled")
				k.SetStringValue("AllowedDomainsForApps", webString)
				k.SetStringValue("RestrictSigninToPattern", profileString)
				k.Close()
			}
		}

		// Firefox
		fk, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, ffPath, registry.ALL_ACCESS)
		if fk != 0 {
			fk.SetStringValue("AllowedDomainsForApps", webString)
			fk.Close()
		}
		fbk, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, ffFilter+`\Block`, registry.ALL_ACCESS)
		if fbk != 0 {
			fbk.SetStringValue("1", "*")
			fbk.Close()
		}
		// Clear old exceptions
		registry.DeleteKey(registry.LOCAL_MACHINE, ffFilter+`\Exceptions`)
		fek, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, ffFilter+`\Exceptions`, registry.ALL_ACCESS)
		if fek != 0 {
			for i, site := range ffExceptions {
				fek.SetStringValue(fmt.Sprintf("%d", i+1), site)
			}
			fek.Close()
		}

		// Restart browsers (kill them so policy takes effect on next launch)
		runPS(`Stop-Process -Name chrome,msedge,brave,opera,firefox -Force -ErrorAction SilentlyContinue`)
		time.Sleep(3 * time.Second)
		return "Browser policy applied and browsers restarted. Policy is now active.", nil

	case "reset-browser-policy":
		chromKeys := []string{
			`SOFTWARE\Policies\Google\Chrome`,
			`SOFTWARE\Policies\Microsoft\Edge`,
			`SOFTWARE\Policies\BraveSoftware\Brave`,
		}
		for _, kPath := range chromKeys {
			k, _ := registry.OpenKey(registry.LOCAL_MACHINE, kPath, registry.ALL_ACCESS)
			if k != 0 {
				k.DeleteValue("AllowedDomainsForApps")
				k.DeleteValue("RestrictSigninToPattern")
				k.Close()
			}
		}
		registry.DeleteKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Mozilla\Firefox`)
		runPS(`Stop-Process -Name chrome,msedge,brave,opera,firefox -Force -ErrorAction SilentlyContinue`)
		time.Sleep(3 * time.Second)
		return "Browser policy reset. Browsers restarted.", nil

	// ── Network tools ─────────────────────────────────────────────

	case "allow-ping":
		cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
			`name=Allow ICMPv4-In`, "protocol=icmpv4:8,any", "dir=in", "action=allow")
		cmd.SysProcAttr = hide
		out, _ := cmd.CombinedOutput()
		return "ICMP Ping allowed.\n" + string(out), nil

	case "ipconfig-flushdns":
		cmd := exec.Command("ipconfig", "/flushdns")
		cmd.SysProcAttr = hide
		out, _ := cmd.CombinedOutput()
		return string(out), nil

	case "ipconfig-renew":
		exec.Command("ipconfig", "/release").Run()
		cmd := exec.Command("ipconfig", "/renew")
		cmd.SysProcAttr = hide
		out, _ := cmd.CombinedOutput()
		return string(out), nil

	case "netsh-winsock-reset":
		cmd := exec.Command("netsh", "winsock", "reset")
		cmd.SysProcAttr = hide
		out, _ := cmd.CombinedOutput()
		return string(out) + "\nReboot required to complete Winsock reset.", nil

	case "netsh-ip-reset":
		cmd := exec.Command("netsh", "int", "ip", "reset")
		cmd.SysProcAttr = hide
		out, _ := cmd.CombinedOutput()
		return string(out) + "\nReboot required to complete TCP/IP reset.", nil

	case "cleanmgr":
		cmd := exec.Command("cleanmgr", "/sagerun:1")
		cmd.SysProcAttr = hide
		cmd.Start()
		return "Windows Disk Cleanup launched.", nil

	case "delete-vss":
		cmd := exec.Command("vssadmin", "delete", "shadows", "/for=c:", "/all", "/quiet")
		cmd.SysProcAttr = hide
		out, _ := cmd.CombinedOutput()
		return "Volume Shadow Copies deleted.\n" + string(out), nil
	}
	return "Action not recognized: " + action, nil
}

func (a *App) GetNativeSetupStatus() (map[string]string, error) {
	status := make(map[string]string)
	h, _ := os.Hostname()
	status["PCName"] = h
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Terminal Server`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		v, _, _ := k.GetIntegerValue("fDenyTSConnections")
		if v == 0 {
			status["RDPStatus"] = "Enabled"
		} else {
			status["RDPStatus"] = "Disabled"
		}
	}
	status["Server121"] = "Disconnected"
	if _, err := os.Stat("Z:"); err == nil {
		status["Server121"] = "Connected"
	}
	status["NASStorage"] = "Disconnected"
	if _, err := os.Stat("Y:"); err == nil {
		status["NASStorage"] = "Connected"
	}
	// Timezone
	cmd := exec.Command("powershell", "-Command", "(Get-TimeZone).DisplayName")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, _ := cmd.CombinedOutput()
	status["TimeZone"] = strings.TrimSpace(string(out))
	return status, nil
}

type SoftwareItem struct {
	ID          string `json:"ID"`
	Name        string `json:"Name"`
	Version     string `json:"Version,omitempty"`
	Category    string `json:"Category,omitempty"`
	SubCategory string `json:"SubCategory,omitempty"`
	Type        string `json:"Type"`
	Package     string `json:"Package,omitempty"`
	Path        string `json:"Path,omitempty"`
	Args        string `json:"Args,omitempty"`
	Description string `json:"Description,omitempty"`
}

func (a *App) GetNativeInstallerCatalog() []SoftwareItem {
	networkPath := `\\174.156.4.3\fjt\Guru`
	return []SoftwareItem{
		// ── BASIC ───────────────────────────────────────────────────
		{ID: "chrome",      Name: "Google Chrome",      Version: "LATEST",  Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "googlechrome", Description: "Fast, secure web browser by Google."},
		{ID: "7zip",        Name: "7-Zip",               Version: "24.08",   Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "7zip", Description: "High compression ratio file archiver."},
		{ID: "winrar",      Name: "WinRAR",              Version: "7.01",    Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "winrar", Description: "Powerful archive manager with RAR format support."},
		{ID: "npp",         Name: "Notepad++",           Version: "8.8.5",   Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "notepadplusplus", Description: "Free source code editor and Notepad replacement."},
		{ID: "vscode",      Name: "VS Code",             Version: "1.102.0", Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "vscode", Description: "Code editing redefined. Built for modern web and cloud."},
		{ID: "vlc",         Name: "VLC Media Player",   Version: "3.0.21",  Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "vlc", Description: "Free and open source multimedia player."},
		{ID: "lightshot",   Name: "Lightshot",           Version: "5.5.0",   Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "lightshot", Description: "Fast screen capture tool with easy sharing."},
		{ID: "zoom",        Name: "Zoom",                Version: "Latest",  Category: "Software install", SubCategory: "Basic", Type: "Choco", Package: "zoom", Description: "Video conferencing and virtual meetings."},

		// ── DEVELOPER ────────────────────────────────────────────────
		{ID: "git",         Name: "Git",                 Version: "2.44.0",  Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "git", Description: "Distributed version control system."},
		{ID: "jdk23",       Name: "Java JDK 23",         Version: "23.0.1",  Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "temurin23", Description: "Eclipse Temurin Java 23 Development Kit."},
		{ID: "nodejs",      Name: "NodeJS",              Version: "20.x",    Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "nodejs", Description: "JavaScript runtime built on Chrome's V8 engine."},
		{ID: "nvm",         Name: "NVM",                 Version: "1.1.12",  Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "nvm", Description: "Node Version Manager for Windows."},
		{ID: "python",      Name: "Python 3",            Version: "3.12",    Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "python", Description: "Python 3 programming language environment."},
		{ID: "postman",     Name: "Postman",             Version: "Latest",  Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "postman", Description: "Complete API development platform."},
		{ID: "mysql",       Name: "MySQL",               Version: "8.0",     Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "mysql", Description: "Open-source relational database management system."},
		{ID: "iis10",       Name: "IIS10 Express",       Version: "10.0",    Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "iisexpress", Description: "Lightweight web server optimized for developers."},
		{ID: "maven",       Name: "Apache Maven",        Version: "3.9.6",   Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "maven", Description: "Build automation tool used primarily for Java."},
		{ID: "filezilla",   Name: "FileZilla",           Version: "3.66.5",  Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "filezilla", Description: "Fast and reliable cross-platform FTP client."},
		{ID: "dbeaver",     Name: "DBeaver",             Version: "24.x",    Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "dbeaver", Description: "Universal database tool for developers and DB admins."},
		{ID: "docker",      Name: "Docker Desktop",      Version: "Latest",  Category: "Software install", SubCategory: "Developer", Type: "Choco", Package: "docker-desktop", Description: "Containerization platform for developers."},

		// ── ENTERPRISE / NAS INSTALLS ────────────────────────────────
		{ID: "office2019",     Name: "MS Office 2019",      Version: "2019",    Category: "Software install", SubCategory: "Enterprise", Type: "Exe",   Path: networkPath + `\Office2019\setup.exe`, Args: "/configure install.xml", Description: "Word, Excel, PowerPoint, Outlook 2019."},
		{ID: "activation",     Name: "Office/Win Activation (KMS)", Version: "KMS", Category: "Software install", SubCategory: "Enterprise", Type: "Exe", Path: networkPath + `\Activation\KMS.exe`, Args: "/S", Description: "Volume activation toolkit for Windows and Office."},
		{ID: "vs2022",         Name: "Visual Studio 2022",  Version: "Enterprise", Category: "Software install", SubCategory: "Enterprise", Type: "Exe", Path: networkPath + `\Visual Studio All\VS 2022\Visual Studio Enterprise.exe`, Args: "--passive --norestart --wait", Description: "Full-featured integrated development environment."},
		{ID: "sql2019",        Name: "SQL Server 2019",     Version: "15.0",    Category: "Software install", SubCategory: "Enterprise", Type: "Exe",   Path: networkPath + `\SQL Server 2019 15.0\SQL2019-SSEI-Expr.exe`, Args: "/Action=Install /IACCEPTSQLSERVERLICENSETERMS /Quiet", Description: "Relational database server by Microsoft."},
		{ID: "sql2022",        Name: "SQL Server 2022",     Version: "16.0",    Category: "Software install", SubCategory: "Enterprise", Type: "Exe",   Path: networkPath + `\SQL Server 2022 16.0\SQL2022-SSEI-Expr.exe`, Args: "/Action=Install /IACCEPTSQLSERVERLICENSETERMS /Quiet", Description: "Latest scalable database server by Microsoft."},
		{ID: "ssms",           Name: "SSMS 20.0",           Version: "20.0",    Category: "Software install", SubCategory: "Enterprise", Type: "Exe",   Path: networkPath + `\SSMS\SSMS-Setup-ENU 2024 20.0.exe`, Args: "/Quiet /NoRestart", Description: "SQL Server Management Studio 20.0."},
		{ID: "mongodb",        Name: "MongoDB",             Version: "7.0.4",   Category: "Software install", SubCategory: "Enterprise", Type: "Msi",   Path: networkPath + `\MongoDB 7.0.4\mongodb-windows-x86_64-7.0.4-signed.msi`, Args: "/quiet /norestart", Description: "NoSQL document-oriented database."},
		{ID: "sqlyog",         Name: "SQL Yog",             Version: "13.2",    Category: "Software install", SubCategory: "Enterprise", Type: "Choco", Package: "sqlyog", Description: "GUI database administration tool for MySQL."},

		// ── SERVICES / MIDDLEWARE ────────────────────────────────────
		{ID: "redis",          Name: "Redis",               Version: "3.0.5",   Category: "Software install", SubCategory: "Enterprise", Type: "Choco", Package: "redis-64", Description: "In-memory data structure store used as database/cache."},
		{ID: "rabbitmq",       Name: "RabbitMQ Server",     Version: "3.11.3",  Category: "Software install", SubCategory: "Enterprise", Type: "Choco", Package: "rabbitmq", Description: "Most widely deployed open source message broker."},
		{ID: "elasticsearch",  Name: "Elasticsearch",       Version: "8.11.1",  Category: "Software install", SubCategory: "Enterprise", Type: "Choco", Package: "elasticsearch", Description: "Distributed, RESTful search and analytics engine."},
		{ID: "erlang",         Name: "Erlang OTP",          Version: "25.1.2",  Category: "Software install", SubCategory: "Enterprise", Type: "Choco", Package: "erlang", Description: "Programming language runtime used by RabbitMQ."},
		{ID: "python2",        Name: "Python 2.7",          Version: "2.7.18",  Category: "Software install", SubCategory: "Enterprise", Type: "Choco", Package: "python2", Description: "Legacy Python 2.7 runtime environment."},
	}
}

// emitLog sends a line to the frontend installer:log event
func (a *App) emitLog(text string, ok bool) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "installer:log", map[string]interface{}{"text": text, "ok": ok})
	}
}

// streamCmd runs a command and streams output as installer:log events.
// Lines are collected from stdout+stderr concurrently via a buffered channel,
// then emitted in batches (max every 150 ms) to avoid flooding the Wails WebView.
func (a *App) streamCmd(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Buffered channel collects lines from both streams without blocking
	lines := make(chan string, 1000)

	scanStream := func(r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 64*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			select {
			case lines <- line:
			default: // drop if channel full rather than block
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go scanStream(stdout, &wg)
	go scanStream(stderr, &wg)

	// Close channel when both streams finish
	go func() {
		wg.Wait()
		close(lines)
	}()

	// Rate-limited batch emitter — flushes at most every 150 ms
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	var batch []string
	emitBatch := func() {
		if len(batch) == 0 {
			return
		}
		for _, l := range batch {
			lLow := strings.ToLower(l)
			ok := !strings.Contains(lLow, "error") &&
				!strings.Contains(lLow, "fail") &&
				!strings.Contains(lLow, "warn")
			a.emitLog(l, ok)
		}
		batch = batch[:0]
	}

	for {
		select {
		case line, open := <-lines:
			if !open {
				// Channel closed — emit remaining batch and exit
				emitBatch()
				goto done
			}
			batch = append(batch, line)
		case <-ticker.C:
			emitBatch()
		}
	}

done:
	return cmd.Wait()
}



// resolveChoco finds the choco.exe path and ensures its bin dir is in the
// current process PATH. Returns the full path to choco.exe.
func resolveChoco() (string, error) {
	// Known Chocolatey install locations (in priority order)
	candidates := []string{
		`C:\ProgramData\chocolatey\bin\choco.exe`,
		`C:\chocolatey\bin\choco.exe`,
	}

	// First check known paths
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			// Found — make sure its parent dir is in PATH so child processes
			// (like choco itself spawning helpers) also work
			binDir := filepath.Dir(p)
			current := os.Getenv("PATH")
			if !strings.Contains(strings.ToLower(current), strings.ToLower(binDir)) {
				os.Setenv("PATH", binDir+";"+current)
			}
			return p, nil
		}
	}

	// Fallback: try PATH lookup
	p, err := exec.LookPath("choco")
	return p, err
}

func (a *App) InstallSoftwareNative(id string) (string, error) {
	catalog := a.GetNativeInstallerCatalog()
	var app SoftwareItem
	found := false
	for _, item := range catalog {
		if item.ID == id {
			app = item
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("software ID %s not found", id)
	}

	// Create a cancellable context and register it
	ctx, cancel := context.WithCancel(context.Background())
	a.installMu.Lock()
	if a.installCancel != nil {
		a.installCancel() // cancel any previous
	}
	a.installCancel = cancel
	a.installMu.Unlock()

	defer func() {
		a.installMu.Lock()
		a.installCancel = nil
		a.installMu.Unlock()
		cancel()
	}()

	a.emitLog("▶ Starting: "+app.Name, true)

	// For NAS-based installs, verify the path is accessible before attempting
	if (app.Type == "Exe" || app.Type == "Msi") && app.Path != "" {
		if _, err := os.Stat(app.Path); os.IsNotExist(err) {
			nasErr := fmt.Sprintf("❌ NAS path not accessible: %s\n   → Make sure the NAS (Z: drive) is connected before installing %s.", app.Path, app.Name)
			a.emitLog(nasErr, false)
			return "", fmt.Errorf("NAS path not found: %s", app.Path)
		}
	}

	switch app.Type {
	case "Choco":
		chocoPath, err := resolveChoco()
		if err != nil {
			// Chocolatey not found anywhere — try to install it
			a.emitLog("⚙️ Chocolatey not found — installing it now...", true)
			installChoco := `Set-ExecutionPolicy Bypass -Scope Process -Force; ` +
				`[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; ` +
				`iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))`
			if err2 := a.streamCmd(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", installChoco); err2 != nil {
				a.emitLog("❌ Failed to install Chocolatey: "+err2.Error(), false)
				return "", fmt.Errorf("choco bootstrap failed: %w", err2)
			}
			// Now resolve again after install
			chocoPath, err = resolveChoco()
			if err != nil {
				a.emitLog("❌ Chocolatey installed but still not found. Please restart the app.", false)
				return "", fmt.Errorf("choco not found after install: %w", err)
			}
			a.emitLog("✅ Chocolatey ready at: "+chocoPath, true)
		} else {
			a.emitLog("✅ Chocolatey found: "+chocoPath, true)
		}

		// --ignore-checksums: Google Chrome & similar apps update frequently;
		// their installer hash often changes before the choco package is updated.
		// --force: clears stale cached downloads.
		chocoArgs := []string{"install", app.Package, "-y", "--no-progress", "--ignore-checksums", "--force"}
		a.emitLog("📦 Running: choco install "+app.Package+" -y --no-progress --ignore-checksums --force", true)
		chocoErr := a.streamCmd(ctx, chocoPath, chocoArgs...)
		if chocoErr != nil {
			if ctx.Err() != nil {
				a.emitLog("⛔ Installation cancelled.", false)
				return "", fmt.Errorf("cancelled")
			}
			// Choco failed — try winget as fallback
			a.emitLog("⚠️ Choco failed — trying winget as fallback...", true)
			wingetArgs := []string{"install", "--id", app.Package,
				"-e", "--silent",
				"--accept-package-agreements", "--accept-source-agreements",
				"--source", "winget",
			}
			if winErr := a.streamCmd(ctx, "winget", wingetArgs...); winErr != nil {
				if ctx.Err() != nil {
					return "", fmt.Errorf("cancelled")
				}
				a.emitLog("❌ Both choco and winget failed for "+app.Name, false)
				return "", fmt.Errorf("choco: %v; winget: %v", chocoErr, winErr)
			}
			a.emitLog("✅ Done via winget: "+app.Name, true)
		} else {
			a.emitLog("✅ Done: "+app.Name, true)
		}
		return "Installed " + app.Name, nil


	case "Exe":
		a.emitLog("▶ Launching installer: "+app.Path, true)
		cmd := exec.CommandContext(ctx, app.Path, strings.Split(app.Args, " ")...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err != nil {
			a.emitLog("❌ Launch failed: "+err.Error(), false)
			return "", fmt.Errorf("exe launch failed: %w", err)
		}
		a.emitLog("⏳ Installer launched — waiting for completion...", true)
		if err := cmd.Wait(); err != nil {
			a.emitLog("⚠️ Installer exited with: "+err.Error(), false)
		}
		a.emitLog("✅ Done: "+app.Name, true)
		return "Launched " + app.Name + " installer", nil

	case "Msi":
		a.emitLog("▶ Running MSI: "+app.Path, true)
		args := append([]string{"/i", app.Path, "/passive", "/norestart"}, strings.Split(app.Args, " ")...)
		if err := a.streamCmd(ctx, "msiexec.exe", args...); err != nil {
			if ctx.Err() != nil {
				a.emitLog("⛔ Installation cancelled.", false)
				return "", fmt.Errorf("cancelled")
			}
			a.emitLog("⚠️ MSI exited with: "+err.Error(), false)
		}
		a.emitLog("✅ Done: "+app.Name, true)
		return "Launched " + app.Name + " installer", nil
	}
	return "Unsupported install type", nil
}

func (a *App) InstallBulkNative(ids []string) (string, error) {
	catalog := a.GetNativeInstallerCatalog()
	
	ctx, cancel := context.WithCancel(context.Background())
	a.installMu.Lock()
	if a.installCancel != nil {
		a.installCancel()
	}
	a.installCancel = cancel
	a.installMu.Unlock()

	defer func() {
		a.installMu.Lock()
		a.installCancel = nil
		a.installMu.Unlock()
		cancel()
	}()

	var chocoApps []SoftwareItem
	var otherApps []SoftwareItem

	for _, id := range ids {
		for _, item := range catalog {
			if item.ID == id {
				if item.Type == "Choco" {
					chocoApps = append(chocoApps, item)
				} else {
					otherApps = append(otherApps, item)
				}
				break
			}
		}
	}

	if len(chocoApps) > 0 {
		chocoPath, err := resolveChoco()
		if err == nil {
			a.emitLog("✅ Chocolatey found: "+chocoPath, true)
			var pkgs []string
			var names []string
			for _, app := range chocoApps {
				pkgs = append(pkgs, app.Package)
				names = append(names, app.Name)
			}
			
			a.emitLog(fmt.Sprintf("▶ Starting bulk install of %d app(s): %s", len(chocoApps), strings.Join(names, ", ")), true)
			
			chocoArgs := append([]string{"install"}, pkgs...)
			chocoArgs = append(chocoArgs, "-y", "--no-progress", "--ignore-checksums", "--force")
			
			a.emitLog("📦 Running: choco install <packages> -y --no-progress", true)
			chocoErr := a.streamCmd(ctx, chocoPath, chocoArgs...)
			if chocoErr != nil {
				a.emitLog("⚠️ Bulk choco install reported an error: "+chocoErr.Error()+". Some packages may have failed.", false)
			} else {
				a.emitLog("✅ Bulk choco install completed.", true)
			}
		} else {
			a.emitLog("❌ Chocolatey not found. Please install it first.", false)
		}
	}

	for _, app := range otherApps {
		a.emitLog("▶ Starting: "+app.Name, true)
		if app.Type == "Exe" {
			cmd := exec.CommandContext(ctx, app.Path, strings.Split(app.Args, " ")...)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if err := cmd.Start(); err != nil {
				a.emitLog("❌ Launch failed: "+err.Error(), false)
			} else {
				cmd.Wait()
				a.emitLog("✅ Done: "+app.Name, true)
			}
		} else if app.Type == "Msi" {
			args := append([]string{"/i", app.Path, "/passive", "/norestart"}, strings.Split(app.Args, " ")...)
			if err := a.streamCmd(ctx, "msiexec.exe", args...); err != nil {
				a.emitLog("⚠️ MSI exited with: "+err.Error(), false)
			} else {
				a.emitLog("✅ Done: "+app.Name, true)
			}
		}
	}
	
	return "Bulk install finished", nil
}


func (a *App) ApplyNativeSecurity(action string, value string) (string, error) {
	switch action {
	case "USB":
		return "USB storage is allowed by default", nil

	case "RDP_Security":
		k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`, registry.SET_VALUE)
		defer k.Close()
		if value == "Block" {
			k.SetDWordValue("fDisableClip", 1)
			k.SetDWordValue("DisableClipboardRedirection", 1)
			k.SetDWordValue("DisableDriveRedirection", 1)
			return "RDP Security Tightened", nil
		} else {
			k.DeleteValue("fDisableClip")
			k.DeleteValue("DisableClipboardRedirection")
			k.DeleteValue("DisableDriveRedirection")
			return "RDP Security Relaxed", nil
		}

	case "BT":
		k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\Bluetooth`, registry.SET_VALUE)
		defer k.Close()
		if value == "Block" {
			k.SetDWordValue("DisableFileTransfer", 1)
			return "Bluetooth Blocked", nil
		} else {
			k.DeleteValue("DisableFileTransfer")
			return "Bluetooth Allowed", nil
		}

	case "Browser":
		for _, key := range []string{`SOFTWARE\Policies\Google\Chrome`, `SOFTWARE\Policies\Microsoft\Edge`} {
			k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
			if value == "" {
				k.DeleteValue("AllowedDomainsForApps")
			} else {
				k.SetStringValue("AllowedDomainsForApps", value)
			}
			k.Close()
		}
		// Restart browsers to apply policy
		for _, browser := range []string{"chrome.exe", "msedge.exe", "firefox.exe"} {
			exec.Command("taskkill", "/F", "/IM", browser, "/T").Run()
		}
		return "Browser Policy Updated (Browsers Restarted)", nil

	case "Defender":
		// value: "On" → enable, "Off" → disable — multi-method approach
		runPS := func(script string) (string, error) {
			cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-NonInteractive", "-Command", script)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			out, err := cmd.CombinedOutput()
			return strings.TrimSpace(string(out)), err
		}
		if value == "Off" {
			script := `
$result = @()
try {
    $tpPath = "HKLM:\SOFTWARE\Microsoft\Windows Defender\Features"
    Set-ItemProperty -Path $tpPath -Name "TamperProtection" -Value 4 -Force -ErrorAction Stop
    Start-Sleep -Milliseconds 500
    $result += "TamperProtection=4"
} catch { $result += "TP: $_" }
try {
    $gpPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection"
    if (-not (Test-Path $gpPath)) { New-Item -Path $gpPath -Force | Out-Null }
    Set-ItemProperty -Path $gpPath -Name "DisableRealtimeMonitoring" -Value 1 -Type DWord -Force -ErrorAction Stop
    Set-ItemProperty -Path $gpPath -Name "DisableBehaviorMonitoring" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    $result += "GP applied"
} catch { $result += "GP: $_" }
try {
    $defPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender"
    if (-not (Test-Path $defPath)) { New-Item -Path $defPath -Force | Out-Null }
    Set-ItemProperty -Path $defPath -Name "DisableAntiSpyware" -Value 1 -Type DWord -Force -ErrorAction Stop
    $result += "AntiSpyware=1"
} catch { $result += "AS: $_" }
try {
    Set-MpPreference -DisableRealtimeMonitoring $true -ErrorAction Stop
    $result += "Set-MpPreference OK"
} catch { $result += "MpPref blocked (disable Tamper Protection manually in Windows Security)" }
Write-Output ($result -join " | ")
`
			out, _ := runPS(script)
			return out, nil
		}
		// Enable
		script := `
$gpPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection"
if (Test-Path $gpPath) {
    Remove-ItemProperty -Path $gpPath -Name "DisableRealtimeMonitoring" -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path $gpPath -Name "DisableBehaviorMonitoring" -ErrorAction SilentlyContinue
}
$defPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender"
Remove-ItemProperty -Path $defPath -Name "DisableAntiSpyware" -ErrorAction SilentlyContinue
$tpPath = "HKLM:\SOFTWARE\Microsoft\Windows Defender\Features"
Set-ItemProperty -Path $tpPath -Name "TamperProtection" -Value 5 -Force -ErrorAction SilentlyContinue
Set-MpPreference -DisableRealtimeMonitoring $false -ErrorAction SilentlyContinue
Start-Service WinDefend -ErrorAction SilentlyContinue
Write-Output "Defender Real-Time Protection ENABLED"
`
		out, _ := runPS(script)
		return out, nil

	case "Firewall":
		// value: "On" → enable, "Off" → disable
		runPS := func(script string) (string, error) {
			cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-NonInteractive", "-Command", script)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			out, err := cmd.CombinedOutput()
			return strings.TrimSpace(string(out)), err
		}
		if value == "Off" {
			out, err := runPS(`Set-NetFirewallProfile -All -Enabled False -ErrorAction Stop; Write-Output "Firewall DISABLED"`)
			if err != nil {
				runPS(`netsh advfirewall set allprofiles state off`)
				return "Firewall DISABLED", nil
			}
			return out, nil
		}
		out, err := runPS(`Set-NetFirewallProfile -All -Enabled True -ErrorAction Stop; Write-Output "Firewall ENABLED"`)
		if err != nil {
			runPS(`netsh advfirewall set allprofiles state on`)
			return "Firewall ENABLED", nil
		}
		return out, nil

	case "Telemetry":
		// value: "Block" → disable telemetry, "Allow" → restore
		runPS := func(script string) {
			cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-NonInteractive", "-Command", script)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd.Run()
		}
		if value == "Block" {
			k, _, kerr := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, registry.ALL_ACCESS)
			if kerr != nil {
				return "", fmt.Errorf("registry error: %v", kerr)
			}
			k.SetDWordValue("AllowTelemetry", 0)
			k.Close()
			runPS(`Stop-Service DiagTrack -Force -ErrorAction SilentlyContinue; Set-Service DiagTrack -StartupType Disabled -ErrorAction SilentlyContinue`)
			return "Telemetry BLOCKED — DiagTrack disabled", nil
		}
		// Allow / Restore
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, registry.ALL_ACCESS)
		if k != 0 {
			k.DeleteValue("AllowTelemetry")
			k.Close()
		}
		runPS(`Set-Service DiagTrack -StartupType Automatic -ErrorAction SilentlyContinue; Start-Service DiagTrack -ErrorAction SilentlyContinue`)
		return "Telemetry RESTORED — DiagTrack re-enabled", nil
	}
	return "Security action not recognized", nil
}

type ScanResult struct {
	Size  string   `json:"size"`
	Count int      `json:"count"`
	Files []string `json:"files"`
	Error string   `json:"error,omitempty"`
}

func dirScan(path string, maxFiles int) (int64, int, []string) {
	var size int64
	var count int
	var files []string
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
			count++
			if len(files) < maxFiles {
				files = append(files, p)
			}
		}
		return nil
	})
	return size, count, files
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/1024/1024/1024)
}

func (a *App) ScanNativeCleanup(mode string) (string, error) {
	var totalSize int64
	var totalCount int
	var targets []string
	var allFiles []string
	res := ScanResult{Files: []string{}}

	if mode == "Temp" {
		targets = []string{os.TempDir(), `C:\Windows\Temp`, `C:\Windows\Prefetch`}
	} else if mode == "Updates" {
		targets = []string{`C:\Windows\SoftwareDistribution\Download`}
	} else if mode == "WinOld" {
		targets = []string{`C:\Windows.old`, `C:\$WINDOWS.~BT`, `C:\$WINDOWS.~WS`}
	} else if mode == "Bin" {
		res.Error = "Recycle Bin & DNS Cache (Scanning requires deeper OS hooks)"
	} else if mode == "SxS" {
		res.Error = "WinSxS Component Store (Deep OS Scan Required)"
	} else if mode == "Bloat" {
		res.Error = "Native Bloatware Packages (Checking installed packages)"
	}

	for _, t := range targets {
		s, c, f := dirScan(t, 100)
		totalSize += s
		totalCount += c
		if len(allFiles) < 100 {
			allFiles = append(allFiles, f...)
		}
	}
	
	if len(allFiles) > 100 {
		res.Files = allFiles[:100]
	} else {
		res.Files = allFiles
	}
	res.Size = formatSize(totalSize)
	res.Count = totalCount

	b, _ := json.Marshal(res)
	return string(b), nil
}

func (a *App) ScanAdvancedCleanup(mode string) (string, error) {
	var totalSize int64
	var totalCount int
	var targets []string
	var allFiles []string
	res := ScanResult{Files: []string{}}

	if mode == "Junk" {
		targets = []string{`C:\Windows\Minidump`, `C:\Windows\Memory.dmp`, os.Getenv("USERPROFILE") + `\AppData\Local\CrashDumps`}
	} else if mode == "Browser" {
		localApp := os.Getenv("LOCALAPPDATA")
		targets = []string{
			filepath.Join(localApp, `Google\Chrome\User Data\Default\Cache`),
			filepath.Join(localApp, `Microsoft\Edge\User Data\Default\Cache`),
			filepath.Join(localApp, `Mozilla\Firefox\Profiles`),
		}
	} else if mode == "WinLogs" {
		targets = []string{`C:\Windows\Logs\CBS`, `C:\Windows\Logs\DISM`, `C:\inetpub\logs\LogFiles`}
	} else if mode == "VSS" {
		res.Error = "Volume Shadow Copies (System allocation scan required)"
	}

	for _, t := range targets {
		s, c, f := dirScan(t, 100)
		totalSize += s
		totalCount += c
		if len(allFiles) < 100 {
			allFiles = append(allFiles, f...)
		}
	}
	
	if len(allFiles) > 100 {
		res.Files = allFiles[:100]
	} else {
		res.Files = allFiles
	}
	res.Size = formatSize(totalSize)
	res.Count = totalCount

	b, _ := json.Marshal(res)
	return string(b), nil
}

func (a *App) RunNativeCleanup(mode string) (string, error) {
	messages := []string{}

	if mode == "Temp" || mode == "Deep" {
		tempFolders := []string{os.TempDir(), `C:\Windows\Temp`}
		for _, f := range tempFolders {
			os.RemoveAll(f)
			messages = append(messages, "Cleaned: "+f)
		}
		// Prefetch etc via cmd
		exec.Command("cmd", "/c", `del /q /f /s C:\Windows\Prefetch\*`).Run()
	}

	if mode == "Updates" || mode == "Deep" {
		exec.Command("net", "stop", "wuauserv").Run()
		os.RemoveAll(`C:\Windows\SoftwareDistribution\Download`)
		exec.Command("net", "start", "wuauserv").Run()
		messages = append(messages, "Update Cache Cleared")
	}

	if mode == "WinOld" || mode == "Deep" {
		// Windows.old is created on C:\ after a Windows feature update.
		// We take ownership first so protected files can be deleted.
		winOldFolders := []string{
			`C:\Windows.old`,
			`C:\$WINDOWS.~BT`,
			`C:\$WINDOWS.~WS`,
		}
		deleted := []string{}
		for _, folder := range winOldFolders {
			if _, err := os.Stat(folder); err == nil {
				// Grant ownership & full access, then remove
				exec.Command("takeown", "/f", folder, "/r", "/d", "Y").Run()
				exec.Command("icacls", folder, "/grant", "Administrators:F", "/t", "/q").Run()
				if remErr := os.RemoveAll(folder); remErr == nil {
					deleted = append(deleted, folder)
				}
			}
		}
		if len(deleted) > 0 {
			messages = append(messages, "Removed: "+strings.Join(deleted, ", "))
		} else {
			messages = append(messages, "Windows.old not found (already clean)")
		}
	}

	if mode == "Bin" || mode == "Deep" {
		exec.Command("powershell", "-NoProfile", "-Command", "Clear-RecycleBin -Force").Run()
		exec.Command("ipconfig", "/flushdns").Run()
		messages = append(messages, "Bin Emptied & DNS Flushed")
	}

	if mode == "SxS" || mode == "Deep" {
		cmd := exec.Command("dism.exe", "/online", "/Cleanup-Image", "/StartComponentCleanup")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Start() // Don't wait, it's too slow
		messages = append(messages, "WinSxS Cleanup Started in Background")
	}

	return strings.Join(messages, ", "), nil
}

func (a *App) ApplyNativeNetworkConfig(adapterUI, ip, mask, gateway, dns string) (string, error) {
	// Resolve real Windows adapter name via netsh
	realAdapter := a.resolveAdapterName(adapterUI)

	// Set static IP / mask / gateway
	// Using name="Adapter Name" directly without additional escaped quotes
	cmd := exec.Command("netsh", "interface", "ipv4", "set", "address",
		"name="+realAdapter, "static", ip, mask, gateway)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to set IP on '%s': %w\n%s", realAdapter, err, string(out))
	}

	// Set primary DNS
	if dns != "" {
		primaryDNS := strings.Split(strings.ReplaceAll(dns, " ", ""), ",")[0]
		dnsCmd := exec.Command("netsh", "interface", "ipv4", "set", "dns",
			"name="+realAdapter, "static", primaryDNS, "primary")
		dnsCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		dnsCmd.Run()
		// Secondary DNS (if provided)
		parts := strings.Split(strings.ReplaceAll(dns, " ", ""), ",")
		if len(parts) > 1 {
			exec.Command("netsh", "interface", "ipv4", "add", "dns",
				"name="+realAdapter, parts[1], "index=2").Run()
		}
	}

	return fmt.Sprintf("Success! Applied on '%s': %s. (Reboot recommended)", realAdapter, ip), nil
}

// resolveAdapterName finds the real Windows interface name matching the UI label.
// Uses PowerShell Get-NetAdapter for reliable adapter identification, filtering out
// VMware, Fortinet, Hyper-V, VPN, TAP, and other virtual adapters by their description.
func (a *App) resolveAdapterName(uiLabel string) string {
	// Use PowerShell to get real adapters with descriptions for accurate filtering
	psCmd := `Get-NetAdapter | Select-Object Name, InterfaceDescription, Status | ForEach-Object { "$($_.Name)|$($_.InterfaceDescription)|$($_.Status)" }`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return uiLabel // fallback
	}

	skipDescPat := []string{"vmware", "virtual", "vpn", "hyper-v", "fortinet", "forticlient",
		"tap", "tunnel", "wan miniport", "loopback", "bluetooth", "teredo"}
	uiLower := strings.ToLower(uiLabel)

	wantEth := strings.Contains(uiLower, "ethernet") || strings.Contains(uiLower, "lan")
	wantWifi := strings.Contains(uiLower, "wi-fi") || strings.Contains(uiLower, "wifi")

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
		if len(parts) < 3 {
			continue
		}
		adapterName := strings.TrimSpace(parts[0])
		description := strings.ToLower(strings.TrimSpace(parts[1]))

		// Skip virtual/VPN adapters by their description
		skip := false
		for _, p := range skipDescPat {
			if strings.Contains(description, p) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		aLower := strings.ToLower(adapterName)
		if wantEth && (strings.Contains(aLower, "ethernet") || strings.Contains(aLower, "local area") || strings.Contains(description, "realtek") || strings.Contains(description, "intel")) {
			return adapterName
		}
		if wantWifi && (strings.Contains(aLower, "wi-fi") || strings.Contains(aLower, "wifi") || strings.Contains(description, "wireless") || strings.Contains(description, "wi-fi")) {
			return adapterName
		}
	}
	return uiLabel // fallback to what UI sent
}

// GetAppVersionNative returns the application version string
func (a *App) GetAppVersionNative() string {
	return "V19.4"
}

// SetDesktopWallpaper sets the system wallpaper to the specified image path
func (a *App) SetDesktopWallpaper(imagePath string) (string, error) {
	if imagePath == "" {
		return "", fmt.Errorf("empty image path")
	}
	// Use PowerShell to set wallpaper reliably
	psScript := fmt.Sprintf(`
		Add-Type -TypeDefinition '
		using System;
		using System.Runtime.InteropServices;
		public class Wallpaper {
			[DllImport("user32.dll", CharSet = CharSet.Auto)]
			public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
		}';
		[Wallpaper]::SystemParametersInfo(20, 0, "%s", 3)
	`, imagePath)

	cmd := exec.Command("powershell", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return "Success! Wallpaper set. (Reboot recommended)", nil
}

func (a *App) PingHost(host string) (string, error) {
	cmd := exec.Command("ping", "-n", "1", host)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), nil // Return output even if error (e.g. timeout)
	}
	return string(out), nil
}

// =============================================
// PERFORMANCE & TWEAKS
// =============================================

// DashboardStatus holds the live state of all dashboard-controllable features
type DashboardStatus struct {
	// CPU / Visual
	AnimationsEnabled     bool   `json:"AnimationsEnabled"`
	TransparencyEnabled   bool   `json:"TransparencyEnabled"`
	GameModeEnabled       bool   `json:"GameModeEnabled"`
	HAGSEnabled           bool   `json:"HAGSEnabled"`
	BackgroundAppsEnabled bool   `json:"BackgroundAppsEnabled"`
	PowerPlan             string `json:"PowerPlan"` // balanced | high | powersaver | ultimate
	// Memory
	PageFileManaged       bool `json:"PageFileManaged"`
	PageFileSizeMB        int  `json:"PageFileSizeMB"`
	SysMainEnabled        bool `json:"SysMainEnabled"`
	MemCompressionEnabled bool `json:"MemCompressionEnabled"`
	PrefetchEnabled       bool `json:"PrefetchEnabled"`
}

func (a *App) GetDashboardStatus() (DashboardStatus, error) {
	var s DashboardStatus

	// Animations
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k.GetIntegerValue("VisualFXSetting")
		s.AnimationsEnabled = (v != 2)
		k.Close()
	} else {
		s.AnimationsEnabled = true
	}

	// Transparency
	k2, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k2.GetIntegerValue("EnableTransparency")
		s.TransparencyEnabled = (v == 1)
		k2.Close()
	} else {
		s.TransparencyEnabled = true
	}

	// Game Mode
	k3, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\GameBar`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k3.GetIntegerValue("AutoGameModeEnabled")
		s.GameModeEnabled = (v == 1)
		k3.Close()
	} else {
		s.GameModeEnabled = false
	}

	// HAGS
	k4, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k4.GetIntegerValue("HwSchMode")
		s.HAGSEnabled = (v == 2)
		k4.Close()
	}

	// Background Apps
	k5, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k5.GetIntegerValue("GlobalUserDisabled")
		s.BackgroundAppsEnabled = (v == 0)
		k5.Close()
	} else {
		s.BackgroundAppsEnabled = true
	}

	// Power Plan
	plan, _ := a.GetCurrentPowerPlan()
	s.PowerPlan = plan

	// PageFile — use WMI to check AutomaticManagedPagefile
	type csPageInfo struct {
		AutomaticManagedPagefile bool
	}
	var csPage []csPageInfo
	if wmiErr := wmi.Query("SELECT AutomaticManagedPagefile FROM Win32_ComputerSystem", &csPage); wmiErr == nil && len(csPage) > 0 {
		s.PageFileManaged = csPage[0].AutomaticManagedPagefile
	} else {
		s.PageFileManaged = true
	}

	if !s.PageFileManaged {
		type pfSetting struct {
			InitialSize uint32
		}
		var pfs []pfSetting
		if wmiErr := wmi.Query("SELECT InitialSize FROM Win32_PageFileSetting", &pfs); wmiErr == nil && len(pfs) > 0 {
			s.PageFileSizeMB = int(pfs[0].InitialSize)
		}
	}

	// SysMain service
	k7, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\SysMain`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k7.GetIntegerValue("Start")
		s.SysMainEnabled = (v == 2) // 2=Automatic, 4=Disabled
		k7.Close()
	} else {
		s.SysMainEnabled = true
	}

	// Prefetch
	k8, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management\PrefetchParameters`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k8.GetIntegerValue("EnablePrefetcher")
		s.PrefetchEnabled = (v != 0)
		k8.Close()
	} else {
		s.PrefetchEnabled = true
	}

	// Memory compression – check via PowerShell (true by default on Win11)
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"(Get-MMAgent).MemoryCompression")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, _ := cmd.Output()
	s.MemCompressionEnabled = strings.TrimSpace(strings.ToLower(string(out))) == "true"

	return s, nil
}

type PerformanceStatus struct {
	AnimationsEnabled         bool `json:"AnimationsEnabled"`
	BackgroundAppsEnabled     bool `json:"BackgroundAppsEnabled"`
	PageFileSizeMB            int  `json:"PageFileSizeMB"`
	PageFileMaxMB             int  `json:"PageFileMaxMB"`
	PageFileManaged           bool `json:"PageFileManaged"`
	GlobalRamOptimizerEnabled bool `json:"GlobalRamOptimizerEnabled"`
	StartupCleanupEnabled     bool `json:"StartupCleanupEnabled"`
}

func (a *App) GetPerformanceStatus() (PerformanceStatus, error) {
	var status PerformanceStatus

	// Animations: read HKCU\Control Panel\Desktop\WindowMetrics "MinAnimate"
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		v, _, _ := k.GetIntegerValue("VisualFXSetting")
		status.AnimationsEnabled = v != 2
	} else {
		status.AnimationsEnabled = true // Default: enabled
	}

	// Background Apps: HKCU\Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications GlobalUserDisabled
	k2, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, registry.QUERY_VALUE)
	if err == nil {
		defer k2.Close()
		v, _, _ := k2.GetIntegerValue("GlobalUserDisabled")
		status.BackgroundAppsEnabled = v == 0
	} else {
		status.BackgroundAppsEnabled = true // Default: enabled
	}

	// PageFile: check WMI
	type csPageInfo struct {
		AutomaticManagedPagefile bool
	}
	var csPage []csPageInfo
	if wmiErr := wmi.Query("SELECT AutomaticManagedPagefile FROM Win32_ComputerSystem", &csPage); wmiErr == nil && len(csPage) > 0 {
		status.PageFileManaged = csPage[0].AutomaticManagedPagefile
	} else {
		status.PageFileManaged = true
	}

	if !status.PageFileManaged {
		type pfSetting struct {
			InitialSize uint32
			MaximumSize uint32
		}
		var pfs []pfSetting
		if wmiErr := wmi.Query("SELECT InitialSize, MaximumSize FROM Win32_PageFileSetting", &pfs); wmiErr == nil && len(pfs) > 0 {
			status.PageFileSizeMB = int(pfs[0].InitialSize)
			status.PageFileMaxMB = int(pfs[0].MaximumSize)
		}
	}

	// Ram Optimizer Task Check
	cmdRam := exec.Command("powershell", "-NoProfile", "-Command", "Get-ScheduledTask -TaskName 'GlobalRamOptimizer' -ErrorAction SilentlyContinue")
	cmdRam.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdRam.Run(); err == nil {
		status.GlobalRamOptimizerEnabled = true
	}

	// Startup Cleanup Task Check
	cmdClean := exec.Command("powershell", "-NoProfile", "-Command", "Get-ScheduledTask -TaskName 'TriveniStartupCleanup' -ErrorAction SilentlyContinue")
	cmdClean.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdClean.Run(); err == nil {
		status.StartupCleanupEnabled = true
	}

	return status, nil
}

func (a *App) SetAnimations(enabled bool) (string, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects`, registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	var setting uint32 = 1 // Let Windows choose
	if !enabled {
		setting = 2 // Best Performance (no animations)
	}
	k.SetDWordValue("VisualFXSetting", setting)

	// Also set via SystemParametersInfo
	if !enabled {
		exec.Command("powershell", "-NoProfile", "-Command",
			"$path = 'HKCU:\\Control Panel\\Desktop'; Set-ItemProperty $path 'UserPreferencesMask' ([byte[]](0x90,0x12,0x03,0x80,0x10,0x00,0x00,0x00))").Run()
	} else {
		exec.Command("powershell", "-NoProfile", "-Command",
			"$path = 'HKCU:\\Control Panel\\Desktop'; Set-ItemProperty $path 'UserPreferencesMask' ([byte[]](0x9E,0x1E,0x07,0x80,0x12,0x00,0x00,0x00))").Run()
	}

	if enabled {
		return "Windows animations enabled", nil
	}
	return "Windows animations disabled (Best Performance)", nil
}

func (a *App) RestoreAnimations() (string, error) {
	return a.SetAnimations(true)
}

// ── Power Plan ────────────────────────────────────────────────────────────────

func (a *App) SetPowerPlan(plan string) (string, error) {
	plans := map[string]string{
		"balanced":   "381b4222-f694-41f0-9685-ff5bb260df2e",
		"high":       "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c",
		"powersaver": "a1841308-3541-4fab-bc81-f71556f20b4a",
		"ultimate":   "e9a42b02-d5df-448d-aa00-03f14749eb61",
	}
	guid, ok := plans[plan]
	if !ok {
		return "", fmt.Errorf("unknown plan: %s", plan)
	}
	cmd := exec.Command("powercfg", "/setactive", guid)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		// Ultimate plan may not exist — try duplicating high perf first
		if plan == "ultimate" {
			exec.Command("powercfg", "-duplicatescheme", guid).Run()
			cmd2 := exec.Command("powercfg", "/setactive", guid)
			cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			cmd2.Run()
		}
	}
	return "Power plan set: " + plan, nil
}

func (a *App) GetCurrentPowerPlan() (string, error) {
	cmd := exec.Command("powercfg", "/getactivescheme")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "balanced", nil
	}
	s := strings.ToLower(string(out))
	if strings.Contains(s, "8c5e7fda") {
		return "high", nil
	}
	if strings.Contains(s, "a1841308") {
		return "powersaver", nil
	}
	if strings.Contains(s, "e9a42b02") {
		return "ultimate", nil
	}
	return "balanced", nil
}

// ── Security Center ───────────────────────────────────────────────────────────

type SecurityCenterStatus struct {
	AV              string `json:"AV"`
	RealtimeEnabled bool   `json:"RealtimeEnabled"`
	FirewallEnabled bool   `json:"FirewallEnabled"`
	UAC             string `json:"UAC"`
	IsAdmin         bool   `json:"IsAdmin"`
	TelemetryLevel  int    `json:"TelemetryLevel"` // 0=blocked, 1=security, 3=full
}

// GetSecurityCenterStatus reads live Defender, Firewall, UAC, Admin and Telemetry state
func (a *App) GetSecurityCenterStatus() (SecurityCenterStatus, error) {
	var s SecurityCenterStatus

	// ── AV name from SecurityCenter2 ─────────────────────────────
	var avList []Win32_AntivirusProduct
	if err := wmi.QueryNamespace("SELECT DisplayName FROM AntivirusProduct", &avList, `root\SecurityCenter2`); err == nil && len(avList) > 0 {
		names := []string{}
		for _, av := range avList {
			names = append(names, av.DisplayName)
		}
		s.AV = strings.Join(names, ", ")
	} else {
		s.AV = "None"
	}

	// ── Defender Real-Time Protection ─────────────────────────────
	// Read via PowerShell Get-MpPreference
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-MpPreference -ErrorAction SilentlyContinue).DisableRealtimeMonitoring")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, _ := cmd.Output()
	val := strings.TrimSpace(strings.ToLower(string(out)))
	// DisableRealtimeMonitoring = False  → realtime IS enabled
	s.RealtimeEnabled = (val == "false")

	// ── Firewall (Standard Profile) ──────────────────────────────
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile`, registry.QUERY_VALUE)
	if err == nil {
		v, _, _ := k.GetIntegerValue("EnableFirewall")
		s.FirewallEnabled = (v == 1)
		k.Close()
	}

	// ── UAC ──────────────────────────────────────────────────────
	k2, err2 := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`, registry.QUERY_VALUE)
	if err2 == nil {
		uac, _, _ := k2.GetIntegerValue("EnableLUA")
		if uac == 1 {
			s.UAC = "Enabled"
		} else {
			s.UAC = "Disabled"
		}
		k2.Close()
	}

	// ── Admin Rights ─────────────────────────────────────────────
	k3, err3 := registry.OpenKey(registry.LOCAL_MACHINE, `SAM`, registry.QUERY_VALUE)
	if err3 == nil {
		s.IsAdmin = true
		k3.Close()
	}

	// ── Telemetry Level ──────────────────────────────────────────
	// 0 = Security (off for non-Enterprise), 1 = Basic, 3 = Full
	k4, err4 := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, registry.QUERY_VALUE)
	if err4 == nil {
		tl, _, _ := k4.GetIntegerValue("AllowTelemetry")
		s.TelemetryLevel = int(tl)
		k4.Close()
	} else {
		s.TelemetryLevel = 3 // default: full
	}

	return s, nil
}

// SecurityCenterAction performs live enable/disable for Defender RT, Firewall, Telemetry
func (a *App) SecurityCenterAction(action string) (string, error) {
	runPS := func(script string) (string, error) {
		cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-NoProfile", "-NonInteractive", "-Command", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	switch action {
	case "defender-rt-enable":
		// Re-enable real-time protection — restore GP policy key and use Set-MpPreference
		script := `
# Remove Group Policy override that disabled real-time monitoring
$gpPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection"
if (Test-Path $gpPath) {
    Remove-ItemProperty -Path $gpPath -Name "DisableRealtimeMonitoring" -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path $gpPath -Name "DisableBehaviorMonitoring" -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path $gpPath -Name "DisableOnAccessProtection" -ErrorAction SilentlyContinue
}
$defPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender"
Remove-ItemProperty -Path $defPath -Name "DisableAntiSpyware" -ErrorAction SilentlyContinue

# Re-enable Tamper Protection
$tpPath = "HKLM:\SOFTWARE\Microsoft\Windows Defender\Features"
Set-ItemProperty -Path $tpPath -Name "TamperProtection" -Value 5 -Force -ErrorAction SilentlyContinue

# Enable via Set-MpPreference
Set-MpPreference -DisableRealtimeMonitoring $false -ErrorAction SilentlyContinue

# Restart Defender service
Start-Service WinDefend -ErrorAction SilentlyContinue

Write-Output "Defender Real-Time Protection ENABLED"
`
		out, _ := runPS(script)
		return out, nil

	case "defender-rt-disable":
		// Multi-method approach to bypass Tamper Protection
		script := `
$result = @()

# Method 1: Disable Tamper Protection via registry first
try {
    $tpPath = "HKLM:\SOFTWARE\Microsoft\Windows Defender\Features"
    $currentTP = (Get-ItemProperty -Path $tpPath -Name "TamperProtection" -ErrorAction SilentlyContinue).TamperProtection
    if ($currentTP -ne 4) {
        Set-ItemProperty -Path $tpPath -Name "TamperProtection" -Value 4 -Force -ErrorAction Stop
        Start-Sleep -Milliseconds 500
        $result += "Tamper Protection disabled"
    }
} catch {
    $result += "Tamper Protection registry: $_"
}

# Method 2: Group Policy registry path (bypasses Tamper Protection on Win10 Pro)
try {
    $gpPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection"
    if (-not (Test-Path $gpPath)) { New-Item -Path $gpPath -Force | Out-Null }
    Set-ItemProperty -Path $gpPath -Name "DisableRealtimeMonitoring" -Value 1 -Type DWord -Force -ErrorAction Stop
    Set-ItemProperty -Path $gpPath -Name "DisableBehaviorMonitoring" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    Set-ItemProperty -Path $gpPath -Name "DisableOnAccessProtection" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    $result += "Group Policy override applied"
} catch {
    $result += "GP registry: $_"
}

# Method 3: Disable AntiSpyware via main Defender policy key
try {
    $defPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows Defender"
    if (-not (Test-Path $defPath)) { New-Item -Path $defPath -Force | Out-Null }
    Set-ItemProperty -Path $defPath -Name "DisableAntiSpyware" -Value 1 -Type DWord -Force -ErrorAction Stop
    $result += "AntiSpyware policy set"
} catch {
    $result += "AntiSpyware policy: $_"
}

# Method 4: Set-MpPreference (may still work after above steps)
try {
    Set-MpPreference -DisableRealtimeMonitoring $true -ErrorAction Stop
    $result += "Set-MpPreference: success"
} catch {
    $result += "Set-MpPreference blocked (Tamper Protection active — go to Windows Security manually)"
}

Write-Output ($result -join " | ")
`
		out, _ := runPS(script)
		return out, nil

	case "firewall-enable":
		out, err := runPS(`Set-NetFirewallProfile -All -Enabled True -ErrorAction Stop; Write-Output "Firewall ENABLED"`)
		if err != nil {
			// fallback netsh
			runPS(`netsh advfirewall set allprofiles state on`)
			return "Firewall ENABLED (netsh)", nil
		}
		return out, nil

	case "firewall-disable":
		out, err := runPS(`Set-NetFirewallProfile -All -Enabled False -ErrorAction Stop; Write-Output "Firewall DISABLED"`)
		if err != nil {
			runPS(`netsh advfirewall set allprofiles state off`)
			return "Firewall DISABLED (netsh)", nil
		}
		return out, nil

	case "telemetry-block":
		k, _, kerr := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, registry.ALL_ACCESS)
		if kerr != nil {
			return "", fmt.Errorf("registry error: %v", kerr)
		}
		k.SetDWordValue("AllowTelemetry", 0)
		k.Close()
		// Also disable DiagTrack service
		runPS(`Stop-Service DiagTrack -Force -ErrorAction SilentlyContinue; Set-Service DiagTrack -StartupType Disabled -ErrorAction SilentlyContinue`)
		return "Telemetry BLOCKED (Level 0) — DiagTrack service disabled", nil

	case "telemetry-restore":
		k, _ := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, registry.ALL_ACCESS)
		if k != 0 {
			k.DeleteValue("AllowTelemetry")
			k.Close()
		}
		// Re-enable DiagTrack
		runPS(`Set-Service DiagTrack -StartupType Automatic -ErrorAction SilentlyContinue; Start-Service DiagTrack -ErrorAction SilentlyContinue`)
		return "Telemetry RESTORED — DiagTrack service re-enabled", nil
	}

	return "Unknown security center action: " + action, nil
}

// ── Per-Feature Performance Tweaks ───────────────────────────────────────────

func (a *App) ApplyPerfTweak(key string, enabled bool) (string, error) {
	onOff := func(b bool) string {
		if b {
			return " enabled"
		}
		return " disabled"
	}
	setBool := func(val uint32) uint32 { return val }
	_ = setBool

	switch key {
	// ── Visual: Transparency
	case "transparency":
		k, _, _ := registry.CreateKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 1
		}
		k.SetDWordValue("EnableTransparency", v)
		return "Transparency" + onOff(enabled), nil

	// ── Visual: Shadows under windows
	case "dropshadow":
		k, _, _ := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 1
		}
		k.SetDWordValue("ListviewShadow", v)
		return "Drop shadows" + onOff(enabled), nil

	// ── Visual: Thumbnail previews in taskbar
	case "thumbpreviews":
		k, _, _ := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 1
		}
		k.SetDWordValue("ExtendedUIHoverTime", v)
		return "Taskbar thumbnails" + onOff(enabled), nil

	// ── Game Mode
	case "gamemode":
		k, _, _ := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\GameBar`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 1
		}
		k.SetDWordValue("AllowAutoGameMode", v)
		k.SetDWordValue("AutoGameModeEnabled", v)
		return "Game Mode" + onOff(enabled), nil

	// ── Hardware-Accelerated GPU Scheduling
	case "hags":
		k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 2
		}
		k.SetDWordValue("HwSchMode", v)
		return "Hardware GPU Scheduling" + onOff(enabled) + ". Reboot required.", nil

	// ── Focus Assist / DND notifications
	case "focusassist":
		k, _, _ := registry.CreateKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\CloudStore\Store\Cache\DefaultAccount\$$windows.data.notifications.quiethours-current`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 1
		}
		k.SetDWordValue("Flags", v)
		return "Focus Assist" + onOff(enabled), nil

	// ── SysMain / Superfetch
	case "sysmain":
		action := "stop"
		startType := uint32(4) // Disabled
		if enabled {
			action = "start"
			startType = 2 // Automatic
		}
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\SysMain`, registry.SET_VALUE)
		if err == nil {
			k.SetDWordValue("Start", startType)
			k.Close()
		}
		sc := exec.Command("sc", action, "SysMain")
		sc.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		sc.Run()
		return "SysMain (Superfetch)" + onOff(enabled), nil

	// ── Memory Compression (PS)
	case "memcompression":
		script := "Disable-MMAgent -MemoryCompression"
		if enabled {
			script = "Enable-MMAgent -MemoryCompression"
		}
		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
		return "Memory Compression" + onOff(enabled), nil

	// ── Large System Cache
	case "largecache":
		k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(0)
		if enabled {
			v = 1
		}
		k.SetDWordValue("LargeSystemCache", v)
		return "Large System Cache" + onOff(enabled), nil

	// ── Trim SSD periodically
	case "trimoptimize":
		cmd := exec.Command("defrag", "C:", "/U", "/V", "/L") // TRIM/optimize
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Start()
		return "Storage TRIM/Optimize started in background", nil

	// ── Prefetch
	case "prefetch":
		k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management\PrefetchParameters`, registry.SET_VALUE)
		defer k.Close()
		v := uint32(3)
		if !enabled {
			v = 0
		}
		k.SetDWordValue("EnablePrefetcher", v)
		k.SetDWordValue("EnableSuperfetch", v)
		return "Prefetch" + onOff(enabled), nil
	}

	return "Unknown tweak: " + key, nil
}

// ── Memory Freeing ────────────────────────────────────────────────────────────

func (a *App) ClearStandbyMemory() (string, error) {
	script := `
$code = @"
using System;
using System.Runtime.InteropServices;
public class MemUtil {
    [DllImport("ntdll.dll")] public static extern int NtSetSystemInformation(int InfoClass, ref int Info, int Length);
    public static string ClearStandbyList() { int v = 4; NtSetSystemInformation(80, ref v, 4); return "Standby list cleared"; }
    public static string ClearWorkingSets() { int v = 1; NtSetSystemInformation(80, ref v, 4); return "Working sets trimmed"; }
}
"@
Add-Type $code -ErrorAction SilentlyContinue
[MemUtil]::ClearStandbyList()
[MemUtil]::ClearWorkingSets()
[System.GC]::Collect()
[System.GC]::WaitForPendingFinalizers()
Write-Output "Done"
`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, _ := cmd.CombinedOutput()
	_ = out
	return "Standby memory and working sets cleared successfully.", nil
}

func (a *App) TrimProcessWorkingSets() (string, error) {
	script := `
Get-Process | Where-Object {$_.WorkingSet64 -gt 10MB} | ForEach-Object {
    try { $_.MinWorkingSet = $_.MinWorkingSet } catch {}
}
Write-Output "Done"
`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
	return "Process working sets trimmed.", nil
}

func (a *App) SetBackgroundApps(enabled bool) (string, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\BackgroundAccessApplications`, registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	var val uint32 = 0
	if !enabled {
		val = 1
	}
	k.SetDWordValue("GlobalUserDisabled", val)
	k.SetDWordValue("BackgroundAppGlobalToggle", val^1)

	if enabled {
		return "Background apps enabled", nil
	}
	return "Background apps disabled", nil
}

func (a *App) RestoreBackgroundApps() (string, error) {
	return a.SetBackgroundApps(true)
}

func (a *App) SetPageFile(initialMB int, maxMB int) (string, error) {
	if initialMB == 0 {
		// System managed
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			"$cs = Get-WmiObject Win32_ComputerSystem; $cs.AutomaticManagedPagefile=$true; $cs.Put()")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
		return "PageFile set to System Managed. Reboot required.", nil
	}

	script := fmt.Sprintf(`
$cs = Get-WmiObject Win32_ComputerSystem
$cs.AutomaticManagedPagefile = $false
$cs.Put() | Out-Null
$pf = Get-WmiObject Win32_PageFileSetting
if ($pf) { $pf.Delete() }
Set-WmiInstance -Class Win32_PageFileSetting -Arguments @{Name="C:\pagefile.sys"; InitialSize=%d; MaximumSize=%d}
`, initialMB, maxMB)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("PageFile set failed: %w", err)
	}
	return fmt.Sprintf("PageFile set to %d MB (Initial) / %d MB (Max). Reboot required.", initialMB, maxMB), nil
}

// =============================================
// STARTUP MANAGER
// =============================================

type StartupApp struct {
	Name    string `json:"Name"`
	Command string `json:"Command"`
	Source  string `json:"Source"`
	Enabled bool   `json:"Enabled"`
}

func (a *App) GetStartupApps() ([]StartupApp, error) {
	var apps []StartupApp

	readStartupKey := func(root registry.Key, path string, source string) {
		k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
		if err != nil {
			return
		}
		defer k.Close()
		names, _ := k.ReadValueNames(-1)
		for _, name := range names {
			val, _, _ := k.GetStringValue(name)
			enabled := true
			displayName := name
			if strings.HasSuffix(name, "_disabled") {
				enabled = false
				displayName = strings.TrimSuffix(name, "_disabled")
			}
			apps = append(apps, StartupApp{
				Name:    displayName,
				Command: val,
				Source:  source,
				Enabled: enabled,
			})
		}
	}

	readStartupKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, "HKCU")
	readStartupKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "HKLM")

	return apps, nil
}

func (a *App) SetStartupApp(name, source string, enabled bool) (string, error) {
	var root registry.Key
	var keyPath string

	if source == "HKCU" {
		root = registry.CURRENT_USER
		keyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	} else {
		root = registry.LOCAL_MACHINE
		keyPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
	}

	k, err := registry.OpenKey(root, keyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("cannot open registry key: %w", err)
	}
	defer k.Close()

	// Try to find the value (enabled or disabled version)
	enabledName := name
	disabledName := name + "_disabled"

	if enabled {
		// Find the disabled version and rename to enabled
		val, _, err := k.GetStringValue(disabledName)
		if err == nil {
			k.SetStringValue(enabledName, val)
			k.DeleteValue(disabledName)
			return name + " startup enabled", nil
		}
		return name + " already enabled", nil
	} else {
		// Find the enabled version and rename to disabled
		val, _, err := k.GetStringValue(enabledName)
		if err == nil {
			k.SetStringValue(disabledName, val)
			k.DeleteValue(enabledName)
			return name + " startup disabled", nil
		}
		return name + " already disabled", nil
	}
}

// =============================================
// EXPANDED CLEANER
// =============================================

func (a *App) RunAdvancedCleanup(mode string) (string, error) {
	messages := []string{}

	if mode == "Junk" {
		// Crash dumps
		crashPath := os.ExpandEnv(`%LOCALAPPDATA%\CrashDumps`)
		os.RemoveAll(crashPath)
		messages = append(messages, "Crash dumps cleared")

		// Thumbnail cache
		thumbCache := os.ExpandEnv(`%LOCALAPPDATA%\Microsoft\Windows\Explorer`)
		entries, _ := os.ReadDir(thumbCache)
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "thumbcache_") {
				os.Remove(thumbCache + "\\" + e.Name())
			}
		}
		messages = append(messages, "Thumbnail cache cleared")

		// Windows Error Reporting
		werPath := os.ExpandEnv(`%APPDATA%\Microsoft\Windows\WER`)
		os.RemoveAll(werPath + "\\ReportArchive")
		os.RemoveAll(werPath + "\\ReportQueue")
		messages = append(messages, "Error reports cleared")

		// Clear Event Logs
		logs := []string{"Application", "System", "Security", "Setup"}
		for _, log := range logs {
			exec.Command("wevtutil", "cl", log).Run()
		}
		messages = append(messages, "Windows Event Logs cleared")
	}

	if mode == "Browser" {
		userProfile := os.Getenv("USERPROFILE")

		// Chrome cache
		chromePaths := []string{
			userProfile + `\AppData\Local\Google\Chrome\User Data\Default\Cache`,
			userProfile + `\AppData\Local\Google\Chrome\User Data\Default\Code Cache`,
			userProfile + `\AppData\Local\Google\Chrome\User Data\Default\GPUCache`,
		}
		for _, p := range chromePaths {
			os.RemoveAll(p)
		}
		messages = append(messages, "Chrome cache cleared")

		// Edge cache
		edgePaths := []string{
			userProfile + `\AppData\Local\Microsoft\Edge\User Data\Default\Cache`,
			userProfile + `\AppData\Local\Microsoft\Edge\User Data\Default\Code Cache`,
			userProfile + `\AppData\Local\Microsoft\Edge\User Data\Default\GPUCache`,
		}
		for _, p := range edgePaths {
			os.RemoveAll(p)
		}
		messages = append(messages, "Edge cache cleared")

		// Firefox cache
		ffProfileBase := userProfile + `\AppData\Local\Mozilla\Firefox\Profiles`
		ffProfiles, err := os.ReadDir(ffProfileBase)
		if err == nil {
			for _, profile := range ffProfiles {
				os.RemoveAll(ffProfileBase + `\` + profile.Name() + `\cache2`)
				os.RemoveAll(ffProfileBase + `\` + profile.Name() + `\startupCache`)
			}
			messages = append(messages, "Firefox cache cleared")
		}
	}

	if mode == "WinLogs" {
		// CBS logs
		os.RemoveAll(`C:\Windows\Logs\CBS`)
		messages = append(messages, "CBS logs cleared")

		// DISM logs
		os.Remove(`C:\Windows\Logs\DISM\dism.log`)
		messages = append(messages, "DISM logs cleared")

		// Windows Update logs
		os.Remove(`C:\Windows\WindowsUpdate.log`)
		messages = append(messages, "Windows Update log cleared")

		// IIS logs (if exist)
		os.RemoveAll(`C:\inetpub\logs\LogFiles`)
		messages = append(messages, "IIS logs cleared (if present)")

		// Minidumps
		os.RemoveAll(`C:\Windows\Minidump`)
		messages = append(messages, "Minidump files cleared")
	}

	if len(messages) == 0 {
		return "No action taken", nil
	}
	return strings.Join(messages, "\n"), nil
}

// =============================================
// DEBLOATER & ADVANCED TWEAKS
// =============================================

type DebloatConfig struct {
	Apps    []string        `json:"Apps"`
	Tweaks  []string        `json:"Tweaks"`
	Options map[string]bool `json:"Options"`
}

func (a *App) ApplyAdvancedDebloat(config DebloatConfig) (string, error) {
	messages := []string{}

	// 1. Restore Point
	if config.Options["restorePoint"] {
		exec.Command("powershell", "-Command", "Checkpoint-Computer -Description 'TGS Debloat Restore' -RestorePointType 'MODIFY_SETTINGS'").Run()
		messages = append(messages, "System Restore Point Created")
	}

	// 2. Remove Apps
	if len(config.Apps) > 0 {
		for _, app := range config.Apps {
			// Basic silent removal command
			cmd := fmt.Sprintf("Get-AppxPackage -Name '%s' -AllUsers | Remove-AppxPackage -ErrorAction SilentlyContinue", app)
			if config.Options["removeFromAllUsers"] {
				cmd = fmt.Sprintf("Get-AppxProvisionedPackage -Online | Where-Object {$_.PackageName -like '*%s*'} | Remove-AppxProvisionedPackage -Online -ErrorAction SilentlyContinue; %s", app, cmd)
			}
			exec.Command("powershell", "-Command", cmd).Run()
		}
		messages = append(messages, fmt.Sprintf("Removed %d selected application(s)", len(config.Apps)))
	}

	// 3. Registry Tweaks
	for _, tweak := range config.Tweaks {
		switch tweak {
		case "telemetry":
			k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, registry.SET_VALUE)
			k.SetDWordValue("AllowTelemetry", 0)
			k.Close()
		case "tips":
			k, _, _ := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`, registry.SET_VALUE)
			k.SetDWordValue("RotatingLockScreenEnabled", 0)
			k.SetDWordValue("SubscribedContent-338387Enabled", 0)
			k.Close()
		case "copilot":
			k, _, _ := registry.CreateKey(registry.CURRENT_USER, `Software\Policies\Microsoft\Windows\WindowsCopilot`, registry.SET_VALUE)
			k.SetDWordValue("TurnOffWindowsCopilot", 1)
			k.Close()
		case "faststartup":
			k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Power`, registry.SET_VALUE)
			if err == nil {
				k.SetDWordValue("HiberbootEnabled", 0)
				k.Close()
			}
		case "extensions":
			k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, registry.SET_VALUE)
			if err == nil {
				k.SetDWordValue("HideFileExt", 0)
				k.Close()
			}
		case "widgets":
			k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Dsh`, registry.SET_VALUE)
			k.SetDWordValue("AllowNewsAndInterests", 0)
			k.Close()
		}
	}
	if len(config.Tweaks) > 0 {
		messages = append(messages, "Applied system-wide tweaks")
	}

	// 4. Restart Explorer
	if config.Options["restartExplorer"] {
		exec.Command("taskkill", "/f", "/im", "explorer.exe").Run()
		exec.Command("cmd", "/c", "start explorer.exe").Start()
		messages = append(messages, "Restarted Windows Explorer")
	}

	return strings.Join(messages, "\n"), nil
}

// =============================================
// SMART OPTIMIZATION TASKS
// =============================================

func (a *App) InstallGlobalRamOptimizer(enable bool) (string, error) {
	taskName := "GlobalRamOptimizer"
	installDir := `C:\GlobalRamOptimization`
	scriptPath := filepath.Join(installDir, "Global-Optimizer.ps1")

	// Always try to clean up the existing task and files first
	exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Unregister-ScheduledTask -TaskName '%s' -Confirm:$false -ErrorAction SilentlyContinue", taskName)).Run()

	if !enable {
		// Stop any running instances
		exec.Command("powershell", "-NoProfile", "-Command", "Get-WmiObject Win32_Process | Where-Object { $_.CommandLine -match 'Global-Optimizer|Global-Optimize-Test' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }").Run()
		os.RemoveAll(installDir)
		return "Global RAM Optimizer Service removed.", nil
	}

	// Create directory
	if err := os.MkdirAll(installDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create install directory: %w", err)
	}

	// The script content from PowerShell
	scriptContent := `
# Global Smart Optimizer (Silent)
if (-not ("MemoryTrimmer" -as [type])) {
    $code = @"
using System;
using System.Runtime.InteropServices;
using System.Diagnostics;
public class MemoryTrimmer {
    [DllImport("psapi.dll")]
    public static extern bool EmptyWorkingSet(IntPtr hProcess);
    public static void TrimProcess(int pid) {
        try {
            Process p = Process.GetProcessById(pid);
            EmptyWorkingSet(p.Handle);
        } catch { }
    }
}
"@
    Add-Type -TypeDefinition $code
}

# Self-Hiding
$wCode = @"
using System; using System.Runtime.InteropServices;
public class WindowManager {
    [DllImport("kernel32.dll")] public static extern IntPtr GetConsoleWindow();
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
}
"@
if (-not ("WindowManager" -as [type])) { Add-Type -TypeDefinition $wCode }
[WindowManager]::ShowWindow([WindowManager]::GetConsoleWindow(), 0)

$Exclusions = @(
    "Idle", "System", "Registry", "smss", "csrss", "wininit", "services", 
    "lsass", "winlogon", "fontdrvhost", "dwm", "Memory Compression", "MsMpEng"
)

while ($true) {
    try {
        # Find High Memory Processes (>100MB)
        $targets = Get-Process -ErrorAction SilentlyContinue | Where-Object { 
            $_.WorkingSet -gt 100MB -and 
            $_.ProcessName -notin $Exclusions 
        }
        
        if ($targets) {
            foreach ($proc in $targets) {
                [MemoryTrimmer]::TrimProcess($proc.Id)
            }
        }
    }
    catch { }
    
    Start-Sleep -Seconds 5
}
`
	// Write the script
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write optimizer script: %w", err)
	}

	// Register Task
	registerCmd := fmt.Sprintf(`
$Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File \"%s\""
$Trigger = New-ScheduledTaskTrigger -AtLogon
$User = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$Principal = New-ScheduledTaskPrincipal -UserId $User -LogonType Interactive -RunLevel Highest
$Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)
Register-ScheduledTask -TaskName "%s" -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Force | Out-Null
Start-ScheduledTask -TaskName "%s"
`, scriptPath, taskName, taskName)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", registerCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to register scheduled task: %w", err)
	}

	return "Global RAM Optimizer Service installed and started.", nil
}

func (a *App) TestRamOptimizer() (string, error) {
	scriptContent := `
# --- Auto-Generated Smart Global RAM Optimizer ---
if (-not ("MemoryTrimmer" -as [type])) {
    $code = @"
using System;
using System.Runtime.InteropServices;
using System.Diagnostics;
public class MemoryTrimmer {
    [DllImport("psapi.dll")]
    public static extern bool EmptyWorkingSet(IntPtr hProcess);
    public static void TrimProcess(int pid) {
        try {
            Process p = Process.GetProcessById(pid);
            EmptyWorkingSet(p.Handle);
        } catch { }
    }
}
"@
    Add-Type -TypeDefinition $code
}

$Exclusions = @(
    "Idle", "System", "Registry", "smss", "csrss", "wininit", "services", "lsass", 
    "winlogon", "fontdrvhost", "dwm", "Memory Compression", "MsMpEng", "taskmgr"
)

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "   GLOBAL SMART RAM OPTIMIZER (TESTING)   " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Target: Processes > 100 MB (Teams, Browsers, etc.)" -ForegroundColor Green
Write-Host "Action: Trim Working Set" -ForegroundColor Green
Write-Host "Excluding: Windows System Processes" -ForegroundColor Gray
Write-Host "Press Ctrl+C to stop." -ForegroundColor Yellow
Write-Host "------------------------------------------" -ForegroundColor Gray

try {
    while ($true) {
        $timestamp = Get-Date -Format "HH:mm:ss"
        $totalFreedCycle = 0
        $trimmedApps = @()

        # Find High Memory Processes
        $targets = Get-Process -ErrorAction SilentlyContinue | Where-Object { 
            $_.WorkingSet -gt 100MB -and 
            $_.ProcessName -notin $Exclusions 
        }

        if ($targets) {
            foreach ($proc in $targets) {
                $before = $proc.WorkingSet
                
                # TRIM
                [MemoryTrimmer]::TrimProcess($proc.Id)
                
                # Measure Savings
                try { $proc.Refresh(); $after = $proc.WorkingSet } catch { $after = $before }
                
                $saved = ($before - $after) / 1MB
                if ($saved -gt 10) {
                    $totalFreedCycle += $saved
                    $trimmedApps += "$($proc.ProcessName) (-$([math]::Round($saved,0))MB)"
                }
            }
        }

        if ($totalFreedCycle -gt 0) {
            Write-Host "[$timestamp] Freed $([math]::Round($totalFreedCycle, 0)) MB" -ForegroundColor Green -NoNewline
            if ($trimmedApps.Count -gt 0) {
                 Write-Host " | $($trimmedApps -join ', ')" -ForegroundColor DarkGray
            } else { Write-Host "" }
        }
        else {
             # Heartbeat to show it's not stuck
             Write-Host "." -NoNewline -ForegroundColor DarkGray
        }
        
        Start-Sleep -Seconds 3
    }
}
catch { 
    Write-Host "\nError in Loop: $_" -ForegroundColor Red
    Start-Sleep -Seconds 5
}
Write-Host "\nPress Enter to exit..." -ForegroundColor Yellow
Read-Host
`

	tempFile := filepath.Join(os.TempDir(), "Global-Optimize-Test.ps1")
	if err := os.WriteFile(tempFile, []byte(scriptContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write test script: %w", err)
	}

	// We want to launch this in a VISIBLE new powershell window so the user sees it running
	cmd := exec.Command("cmd", "/c", "start", "powershell.exe", "-NoExit", "-ExecutionPolicy", "Bypass", "-File", tempFile)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // hide the cmd window, but powershell will pop up via start
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to launch test script: %w", err)
	}

	return "RAM Optimizer Test started in a new window.", nil
}

func (a *App) InstallStartupCleanup(enable bool) (string, error) {
	taskName := "TriveniStartupCleanup"
	installDir := `C:\ProgramData\Triveni`
	scriptPath := filepath.Join(installDir, "cleanup_optimized.bat")

	// Unregister existing task and remove files
	exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Unregister-ScheduledTask -TaskName '%s' -Confirm:$false -ErrorAction SilentlyContinue", taskName)).Run()
	os.Remove(scriptPath)

	// Remove legacy startup script if it exists
	startupDir := os.ExpandEnv(`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup`)
	legacyPath := filepath.Join(startupDir, "cleanup_optimized.bat")
	os.Remove(legacyPath)

	if !enable {
		return "Startup Cleanup Task removed.", nil
	}

	if err := os.MkdirAll(installDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create install directory: %w", err)
	}

	scriptContent := `
@echo off
:: Note: This script is run via Scheduled Task with Highest Privileges, so no self-elevation check is needed.

:: ===============================
:: SYSTEM CLEANUP & OPTIMIZATION
:: ===============================

:: Clean User Temp folder completely
echo Cleaning User Temp folder...
rd /s /q "%temp%"
md "%temp%"

:: Clean Windows Temp folder completely
echo Cleaning Windows Temp folder...
rd /s /q "C:\Windows\Temp"
md "C:\Windows\Temp"

:: Delete Prefetch files
echo Cleaning Prefetch files...
del /s /q C:\Windows\Prefetch\*.* >nul 2>&1

:: Optimize Drives (Defrag for HDD, Trim for SSD)
echo Optimizing drive...
defrag C: /O

:: Flush DNS Cache
echo Flushing DNS cache...
ipconfig /flushdns

:: Save Startup Programs list (for manual review)
echo Exporting startup program list to Desktop...
wmic startup get caption,command > "%USERPROFILE%\Desktop\Startup_Programs.txt"

:: Clear Recycle Bin
echo Clearing Recycle Bin...
rd /s /q C:\$Recycle.Bin >nul 2>&1

:: Run Disk Cleanup silently (if pre-configured with cleanmgr /sageset:1)
echo Running Disk Cleanup...
cleanmgr /sagerun:1

:: ===============================
:: FINISH
:: ===============================
echo.
echo ==========================================
echo All cleanup & optimization tasks completed!
echo ==========================================
:: No pause, exit immediately for silent operation
exit
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write cleanup script: %w", err)
	}

	// Register Task
	registerCmd := fmt.Sprintf(`
$Action = New-ScheduledTaskAction -Execute "cmd.exe" -Argument "/c \"%s\""
$Trigger = New-ScheduledTaskTrigger -AtLogon
$User = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$Principal = New-ScheduledTaskPrincipal -UserId $User -LogonType Interactive -RunLevel Highest
Register-ScheduledTask -TaskName "%s" -Action $Action -Trigger $Trigger -Principal $Principal -Force | Out-Null
`, scriptPath, taskName)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", registerCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to register scheduled task: %w", err)
	}

	return "Startup Cleanup Task installed.", nil
}

func (a *App) GetInstalledSoftwareStatusNative() (map[string]bool, error) {
	statusMap := make(map[string]bool)
	catalog := a.GetNativeInstallerCatalog()

	// Direct filesystem binary paths (Instant & 100% accurate)
	searchBases := []string{
		`C:\Program Files`,
		`C:\Program Files (x86)`,
		filepath.Join(os.Getenv("LocalAppData"), "Programs"),
		filepath.Join(os.Getenv("AppData"), "Programs"),
		filepath.Join(os.Getenv("ProgramData"), "chocolatey", "bin"),
		filepath.Join(os.Getenv("SystemDrive") + `\`),
	}

	knownFiles := map[string][]string{
		"chrome":        {`Google\Chrome\Application\chrome.exe`},
		"7zip":          {`7-Zip\7zFM.exe`, `7-Zip\7z.exe`},
		"winrar":        {`WinRAR\WinRAR.exe`, `WinRAR\Rar.exe`},
		"npp":           {`Notepad++\notepad++.exe`},
		"vscode":        {`Microsoft VS Code\Code.exe`, `Microsoft VS Code\bin\code.cmd`},
		"vlc":           {`VideoLAN\VLC\vlc.exe`},
		"lightshot":     {`Skillbrains\lightshot\Lightshot.exe`, `Skillbrains\lightshot\Lightshot.dll`},
		"zoom":          {`Zoom\bin\Zoom.exe`, `Zoom\bin\ZoomHost.exe`},
		"git":           {`Git\bin\git.exe`, `Git\cmd\git.exe`},
		"jdk23":         {`Java\jdk-23\bin\java.exe`, `Java\jdk-23.0.1\bin\java.exe`, `Eclipse Adoptium\jdk-23.0.1.11-hotspot\bin\java.exe`, `Eclipse Adoptium\jdk-23\bin\java.exe`, `Eclipse Foundation\jdk-23\bin\java.exe`},
		"nodejs":        {`nodejs\node.exe`, `nodejs\npm.cmd`},
		"nvm":           {`nvm\nvm.exe`, `nvm4w\nvm.exe`},
		"python":        {`Python313\python.exe`, `Python312\python.exe`, `Python311\python.exe`, `Python310\python.exe`, `Python39\python.exe`},
		"python2":       {`Python27\python.exe`},
		"postman":       {`Postman\Postman.exe`, `Postman\app\Postman.exe`},
		"mysql":         {`MySQL\MySQL Server 8.0\bin\mysqld.exe`, `MySQL\MySQL Server 8.4\bin\mysqld.exe`},
		"iis10":         {`IIS Express\iisexpress.exe`},
		"maven":         {`apache-maven\bin\mvn.cmd`, `Maven\bin\mvn.cmd`},
		"filezilla":     {`FileZilla FTP Client\filezilla.exe`, `FileZilla\filezilla.exe`},
		"dbeaver":       {`DBeaver\dbeaver.exe`},
		"docker":        {`Docker\Docker\resources\bin\docker.exe`, `Docker\Docker\Docker Desktop.exe`},
		"office2019":    {`Microsoft Office\root\Office16\WINWORD.EXE`, `Microsoft Office\Office16\WINWORD.EXE`},
		"vs2022":        {`Microsoft Visual Studio\2022\Enterprise\Common7\IDE\devenv.exe`, `Microsoft Visual Studio\2022\Community\Common7\IDE\devenv.exe`, `Microsoft Visual Studio\2022\Professional\Common7\IDE\devenv.exe`},
		"sql2019":       {`Microsoft SQL Server\MSSQL15.MSSQLSERVER\MSSQL\Binn\sqlservr.exe`, `Microsoft SQL Server\MSSQL15.SQLEXPRESS\MSSQL\Binn\sqlservr.exe`},
		"sql2022":       {`Microsoft SQL Server\MSSQL16.MSSQLSERVER\MSSQL\Binn\sqlservr.exe`, `Microsoft SQL Server\MSSQL16.SQLEXPRESS\MSSQL\Binn\sqlservr.exe`},
		"ssms":          {`Microsoft SQL Server Management Studio 20\Common7\IDE\Ssms.exe`, `Microsoft SQL Server Management Studio 19\Common7\IDE\Ssms.exe`, `Microsoft SQL Server Management Studio 18\Common7\IDE\Ssms.exe`},
		"mongodb":       {`MongoDB\Server\7.0\bin\mongod.exe`, `MongoDB\Server\6.0\bin\mongod.exe`},
		"sqlyog":        {`SQLyog\SQLyog.exe`, `SQLyog\SQLyog Community.exe`},
		"redis":         {`Redis\redis-server.exe`, `Redis\redis-cli.exe`},
		"rabbitmq":      {`RabbitMQ Server\rabbitmq_server-3.11.3\sbin\rabbitmqctl.bat`},
		"elasticsearch": {`Elastic\Elasticsearch\8.11.1\bin\elasticsearch-service.bat`, `Elasticsearch\bin\elasticsearch.bat`},
		"erlang":        {`erl-25.1.2\bin\erl.exe`, `erl-26.0\bin\erl.exe`, `erl-27.0\bin\erl.exe`, `erl25.1.2\bin\erl.exe`},
	}

	// 1) Check direct files first
	for _, item := range catalog {
		if relList, ok := knownFiles[item.ID]; ok {
			for _, base := range searchBases {
				for _, rel := range relList {
					checkPath := filepath.Join(base, rel)
					if _, err := os.Stat(checkPath); err == nil {
						statusMap[item.ID] = true
						break
					}
				}
				if statusMap[item.ID] {
					break
				}
			}
		}
	}

	// 2) Service check for middleware (RabbitMQ, ElasticSearch)
	for _, svcCheck := range []struct {
		id      string
		service string
	}{
		{"rabbitmq", "RabbitMQ"},
		{"elasticsearch", "elasticsearch"},
		{"mysql", "MySQL"},
		{"redis", "Redis"},
	} {
		if !statusMap[svcCheck.id] {
			cmd := exec.Command("sc", "query", svcCheck.service)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := cmd.CombinedOutput(); err == nil && strings.Contains(strings.ToUpper(string(out)), "RUNNING") {
				statusMap[svcCheck.id] = true
			}
		}
	}

	// 3) Fallback to Choco and Registry for any items not yet found
	needRegistry := false
	for _, item := range catalog {
		if !statusMap[item.ID] {
			needRegistry = true
			break
		}
	}

	if needRegistry {
		// Choco list
		chocoOut := ""
		for _, chocoArgs := range [][]string{{"list"}, {"list", "-lo"}} {
			cmdChoco := exec.Command("choco", chocoArgs...)
			cmdChoco.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			out, err := cmdChoco.CombinedOutput()
			if err == nil {
				chocoOut = strings.ToLower(string(out))
				break
			}
		}

		// Registry check
		psCmd := `$keys = @(
  'HKLM:\Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*'
)
Get-ItemProperty $keys -ErrorAction SilentlyContinue | Where-Object {$_.DisplayName} | Select-Object -ExpandProperty DisplayName`
		cmdPs := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
		cmdPs.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out2, _ := cmdPs.CombinedOutput()
		winOut := strings.ToLower(string(out2))

		registryAlias := map[string]string{
			"office2019": "microsoft office",
			"vs2022":     "visual studio",
			"ssms":       "sql server management studio",
			"chrome":     "google chrome",
			"vscode":     "visual studio code",
			"7zip":       "7-zip",
			"npp":        "notepad++",
			"jdk23":      "eclipse temurin",
			"nodejs":     "node.js",
			"python":     "python",
			"mysql":      "mysql",
			"docker":     "docker desktop",
			"mongodb":    "mongodb",
			"dbeaver":    "dbeaver",
			"filezilla":  "filezilla",
			"maven":      "apache maven",
			"redis":      "redis",
			"rabbitmq":   "rabbitmq",
			"elasticsearch": "elasticsearch",
			"sql2019":    "sql server 2019",
			"sql2022":    "sql server 2022",
			"iis10":      "iis",
			"winrar":     "winrar",
			"git":        "git",
			"vlc":        "vlc",
			"postman":    "postman",
			"nvm":        "nvm",
			"zoom":       "zoom",
		}

		for _, item := range catalog {
			if statusMap[item.ID] {
				continue
			}
			searchName := strings.ToLower(item.Name)
			if alias, ok := registryAlias[item.ID]; ok {
				searchName = alias
			}
			if item.Package != "" {
				pkg := strings.ToLower(item.Package)
				if strings.Contains(chocoOut, pkg+" ") || strings.Contains(chocoOut, pkg+"\r") || strings.Contains(chocoOut, pkg+"\n") {
					statusMap[item.ID] = true
					continue
				}
			}
			if strings.Contains(winOut, searchName) {
				statusMap[item.ID] = true
			}
		}
	}

	return statusMap, nil
}

func (a *App) UninstallSoftwareNative(id string) (string, error) {
	catalog := a.GetNativeInstallerCatalog()
	var app SoftwareItem
	for _, item := range catalog {
		if item.ID == id {
			app = item
			break
		}
	}
	if app.ID == "" {
		return "", fmt.Errorf("software %s not found", id)
	}

	// Create cancellable context for this uninstall
	ctx, cancel := context.WithCancel(context.Background())
	a.installMu.Lock()
	if a.installCancel != nil {
		a.installCancel()
	}
	a.installCancel = cancel
	a.installMu.Unlock()
	defer func() {
		a.installMu.Lock()
		a.installCancel = nil
		a.installMu.Unlock()
		cancel()
	}()

	a.emitLog("🗑️ Removing: "+app.Name, true)

	if app.Type == "Choco" && app.Package != "" {
		chocoPath, err := resolveChoco()
		if err != nil {
			a.emitLog("❌ Chocolatey not found — cannot uninstall via choco.", false)
			return "", fmt.Errorf("choco not found: %w", err)
		}
		a.emitLog("📦 Running: choco uninstall "+app.Package+" -y --remove-dependencies", true)
		if err := a.streamCmd(ctx, chocoPath, "uninstall", app.Package, "-y", "--remove-dependencies"); err != nil {
			if ctx.Err() != nil {
				a.emitLog("⛔ Removal cancelled.", false)
				return "", fmt.Errorf("cancelled")
			}
			a.emitLog("❌ Uninstall failed: "+err.Error(), false)
			return "", fmt.Errorf("choco uninstall failed: %w", err)
		}
		a.emitLog("✅ Removed: "+app.Name, true)
		return "Uninstalled " + app.Name, nil
	}

	a.emitLog("⚠️ "+app.Name+" requires manual removal via Control Panel.", false)
	return "Manual uninstall required for " + app.Name, nil
}

// DeepRemoveSoftwareNative removes software AND cleans leftover AppData/registry entries
func (a *App) DeepRemoveSoftwareNative(id string) (string, error) {
	// First do the standard uninstall
	result, err := a.UninstallSoftwareNative(id)
	if err != nil {
		return result, err
	}

	a.emitLog("🧹 Running deep clean for: "+id, true)

	// Per-app cleanup map: AppData dirs and registry keys to nuke
	type cleanupSpec struct {
		AppDataDirs []string // relative to %APPDATA% or %LOCALAPPDATA%
		ProgramDirs []string // absolute or relative to %ProgramFiles%
		RegKeys     []string // HKCU/HKLM registry paths to delete
	}

	cleanupMap := map[string]cleanupSpec{
		"chrome":    {AppDataDirs: []string{`Google\Chrome`}, RegKeys: []string{`HKCU:\Software\Google\Chrome`}},
		"firefox":   {AppDataDirs: []string{`Mozilla\Firefox`}, RegKeys: []string{`HKCU:\Software\Mozilla`}},
		"vscode":    {AppDataDirs: []string{`Code`, `Code - Insiders`}, RegKeys: []string{`HKCU:\Software\Microsoft\VSCode`}},
		"nodejs":    {AppDataDirs: []string{`npm`, `npm-cache`}, ProgramDirs: []string{`C:\Program Files\nodejs`}},
		"nvm":       {AppDataDirs: []string{`nvm`}, ProgramDirs: []string{`C:\ProgramData\nvm`}},
		"python":    {AppDataDirs: []string{`Python`, `pip`}, RegKeys: []string{`HKCU:\Software\Python`}},
		"mysql":     {AppDataDirs: []string{`MySQL`}, ProgramDirs: []string{`C:\Program Files\MySQL`}, RegKeys: []string{`HKLM:\SOFTWARE\MySQL AB`}},
		"mongodb":   {AppDataDirs: []string{`MongoDB`}, ProgramDirs: []string{`C:\Program Files\MongoDB`}, RegKeys: []string{`HKLM:\SOFTWARE\MongoDB`}},
		"docker":    {AppDataDirs: []string{`Docker`, `Docker Desktop`}, RegKeys: []string{`HKCU:\Software\Docker Inc.`}},
		"postman":   {AppDataDirs: []string{`Postman`}, RegKeys: []string{`HKCU:\Software\Postman`}},
		"dbeaver":   {AppDataDirs: []string{`DBeaverData`}},
		"winrar":    {AppDataDirs: []string{`WinRAR`}, RegKeys: []string{`HKCU:\Software\WinRAR`}},
		"vlc":       {AppDataDirs: []string{`vlc`}},
		"git":       {AppDataDirs: []string{`Git`}, RegKeys: []string{`HKCU:\Software\GitForWindows`}},
		"filezilla": {AppDataDirs: []string{`FileZilla`}},
		"redis":     {ProgramDirs: []string{`C:\Program Files\Redis`}, RegKeys: []string{`HKLM:\SOFTWARE\Redis`}},
		"rabbitmq":  {ProgramDirs: []string{`C:\Program Files\RabbitMQ Server`}, AppDataDirs: []string{`RabbitMQ`}},
	}

	spec, hasSpec := cleanupMap[id]
	if !hasSpec {
		a.emitLog("ℹ️ No extra cleanup defined for: "+id, true)
		return result + " (no extra cleanup)", nil
	}

	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")
	cleaned := 0

	// Remove AppData directories (check both Roaming and Local)
	for _, rel := range spec.AppDataDirs {
		for _, base := range []string{appData, localAppData} {
			full := filepath.Join(base, rel)
			if _, err2 := os.Stat(full); err2 == nil {
				if rem := os.RemoveAll(full); rem == nil {
					a.emitLog("🗑️ Deleted: "+full, true)
					cleaned++
				} else {
					a.emitLog("⚠️ Could not delete: "+full+" — "+rem.Error(), false)
				}
			}
		}
	}

	// Remove Program directories
	for _, dir := range spec.ProgramDirs {
		if _, err2 := os.Stat(dir); err2 == nil {
			if rem := os.RemoveAll(dir); rem == nil {
				a.emitLog("🗑️ Deleted: "+dir, true)
				cleaned++
			} else {
				a.emitLog("⚠️ Could not delete: "+dir+" — "+rem.Error(), false)
			}
		}
	}

	// Remove registry keys via PowerShell
	for _, key := range spec.RegKeys {
		psScript := fmt.Sprintf(`Remove-Item -Path "%s" -Recurse -Force -ErrorAction SilentlyContinue; Write-Output "Reg cleaned: %s"`, key, key)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, _ := cmd.CombinedOutput()
		line := strings.TrimSpace(string(out))
		if line != "" {
			a.emitLog(line, true)
			cleaned++
		}
	}

	a.emitLog(fmt.Sprintf("✅ Deep clean done — %d items removed.", cleaned), true)
	return fmt.Sprintf("Removed %s + %d cleanup items", id, cleaned), nil
}
