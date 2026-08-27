package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed scripts
var scripts embed.FS

//go:embed assets/tgswall.png
var triveniWallBytes []byte

// App struct
type App struct {
	ctx          context.Context
	cachedAudit  string
	auditMutex   sync.Mutex
	installCancel context.CancelFunc // cancels the running install
	installMu    sync.Mutex          // guards installCancel
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Pre-cache audit data in background
	go a.RefreshAudit()
}

// RefreshAudit runs the native audit and caches the result
func (a *App) RefreshAudit() {
	data, err := a.GetNativeAudit()
	if err == nil {
		res, _ := json.Marshal(data)
		a.auditMutex.Lock()
		a.cachedAudit = string(res)
		a.auditMutex.Unlock()
	}
}

// GetAudit returns the hardware audit data
func (a *App) GetAudit() (AuditData, error) {
	return a.GetNativeAudit()
}

// GetStats returns real-time system stats
func (a *App) GetStats() (StatsData, error) {
	return a.GetNativeStats()
}

// RunNativeAction handles setup actions natively
func (a *App) RunNativeAction(action string, args []string) (string, error) {
	return a.ApplyNativeAction(action, args)
}

// GetSetupStatus returns the native setup status
func (a *App) GetSetupStatus() (map[string]string, error) {
	return a.GetNativeSetupStatus()
}

// GetInstallerCatalog returns the built-in software list
func (a *App) GetInstallerCatalog() []SoftwareItem {
	return a.GetNativeInstallerCatalog()
}

// InstallSoftware installs software with real-time event streaming
func (a *App) InstallSoftware(id string) (string, error) {
	return a.InstallSoftwareNative(id)
}

// StopInstall cancels the currently running install/uninstall operation
func (a *App) StopInstall() {
	a.installMu.Lock()
	defer a.installMu.Unlock()
	if a.installCancel != nil {
		a.installCancel()
		a.installCancel = nil
	}
}

// GetInstalledSoftwareStatus returns a map of installed software IDs
func (a *App) GetInstalledSoftwareStatus() (map[string]bool, error) {
	return a.GetInstalledSoftwareStatusNative()
}

// UninstallSoftware uninstalls software with real-time event streaming
func (a *App) UninstallSoftware(id string) (string, error) {
	return a.UninstallSoftwareNative(id)
}

// DeepRemoveSoftware fully removes software including AppData, registry leftovers
func (a *App) DeepRemoveSoftware(id string) (string, error) {
	return a.DeepRemoveSoftwareNative(id)
}

// ApplySecurity updates system security policies
func (a *App) ApplySecurity(action string, value string) (string, error) {
	return a.ApplyNativeSecurity(action, value)
}

// RunCleanup performs system cleanup tasks
func (a *App) RunCleanup(mode string) (string, error) {
	return a.RunNativeCleanup(mode)
}

// ApplyNetworkConfig applies static IP and DNS settings
func (a *App) ApplyNetworkConfig(adapter, ip, mask, gateway, dns string) (string, error) {
	return a.ApplyNativeNetworkConfig(adapter, ip, mask, gateway, dns)
}

// Ping checks connectivity to a host
func (a *App) Ping(host string) (string, error) {
	return a.PingHost(host)
}

// GetCachedAudit returns the last cached audit data
func (a *App) GetCachedAudit() string {
	a.auditMutex.Lock()
	defer a.auditMutex.Unlock()
	return a.cachedAudit
}

// SelectWallpaperFile opens a file dialog to pick an image
func (a *App) SelectWallpaperFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Wallpaper Image",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images (*.png;*.jpg;*.jpeg;*.bmp)", Pattern: "*.png;*.jpg;*.jpeg;*.bmp"},
		},
	})
}

// SetWallpaper sets the selected image as desktop background
func (a *App) SetWallpaper(path string) (string, error) {
	return a.SetDesktopWallpaper(path)
}

// SetDefaultWallpaper extracts the embedded Triveni wallpaper to %TEMP% and applies it
func (a *App) SetDefaultWallpaper() (string, error) {
	// Write embedded bytes to a temp file
	tmpPath := filepath.Join(os.TempDir(), "triveni_wall.png")
	if err := os.WriteFile(tmpPath, triveniWallBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write wallpaper temp file: %w", err)
	}
	return a.SetDesktopWallpaper(tmpPath)
}

// GetAppVersion returns the current version
func (a *App) GetAppVersion() (string, error) {
	return a.GetAppVersionNative(), nil
}


// ToggleAnimations enables or disables Windows visual animations
func (a *App) ToggleAnimations(enabled bool) (string, error) {
	return a.SetAnimations(enabled)
}

// ToggleBackgroundApps enables or disables background apps globally
func (a *App) ToggleBackgroundApps(enabled bool) (string, error) {
	return a.SetBackgroundApps(enabled)
}

// ConfigurePageFile sets the page file size in MB (0 = system managed)
func (a *App) ConfigurePageFile(initialMB int, maxMB int) (string, error) {
	return a.SetPageFile(initialMB, maxMB)
}

// ToggleStartupApp enables or disables a specific startup app
func (a *App) ToggleStartupApp(name, source string, enabled bool) (string, error) {
	return a.SetStartupApp(name, source, enabled)
}

// RunAdvancedClean performs advanced cleanup operations
func (a *App) RunAdvancedClean(mode string) (string, error) {
	return a.RunAdvancedCleanup(mode)
}

// UploadAuditData uploads the provided audit payload to the configured Google endpoint
func (a *App) UploadAuditData(payload map[string]interface{}) (string, error) {
	url := "https://script.google.com/macros/s/AKfycbz6Q-tcoC7hag43zyW0UzyCKbJ_m_EkB5c1IBo5kDC082kOKburY0DabWD92g_vd3JP/exec"

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	return "Success! Data has been uploaded to Google Sheets.", nil
}

// PerformSilentAudit gathers data and uploads it to Google Sheets without UI
func (a *App) PerformSilentAudit() error {
	data, err := a.GetNativeAudit()
	if err != nil {
		return fmt.Errorf("audit failed: %v", err)
	}

	appData := os.Getenv("APPDATA")
	userPath := filepath.Join(appData, "TGS_AssignedUser.txt")
	assignedUser := "Unassigned"
	if b, err := os.ReadFile(userPath); err == nil {
		assignedUser = string(bytes.TrimSpace(b))
	}

	// Flatten data for the Google Script (matching PowerShell format + Audit.svelte)
	payload := map[string]interface{}{
		"Targets":        []string{"SCHEDULE Audit"},
		"Timestamp":      dtString(),
		"PCName":         data.System.Name,
		"AssignedUser":   assignedUser,
		"Processor":      data.Hardware.CPU,
		"RAM":            data.Hardware.RAMTotal,
		"IP":             data.Network.IP,
		"USBStatus":      data.Security.USBStatus,
		"RDPStatus":      data.Security.RDPStatus,
		"BrowserDomain":  data.Security.BrowserPolicy,
		"ScheduleStatus": "ACTIVE",
		"TTCode":         "Scheduled Success",
	}

	// Add storage info (first two drives)
	if len(data.Hardware.Storage) > 0 {
		payload["Storage1"] = data.Hardware.Storage[0].Label
	}
	if len(data.Hardware.Storage) > 1 {
		payload["Storage2"] = data.Hardware.Storage[1].Label
	}

	_, err = a.UploadAuditData(payload)
	return err
}

func dtString() string {
	// returns yyyy-MM-dd HH:mm:ss
	return time.Now().Format("2006-01-02 15:04:05")
}

// ── Dashboard Network Tools ───────────────────────────────────────────────────
// These are exposed via Wails to the frontend Dashboard network tab.

func (a *App) FlushDNS() (string, error) {
	return a.ApplyNativeAction("flush-dns", nil)
}

func (a *App) RenewIP() (string, error) {
	return a.ApplyNativeAction("renew-ip", nil)
}

func (a *App) WinsockReset() (string, error) {
	return a.ApplyNativeAction("winsock-reset", nil)
}

func (a *App) TCPIPReset() (string, error) {
	return a.ApplyNativeAction("tcpip-reset", nil)
}

func (a *App) AllowICMPPing() (string, error) {
	return a.ApplyNativeAction("allow-icmp", nil)
}

// HibernateHeavyApps suspends heavy programs for Game Mode
func (a *App) HibernateHeavyApps() (string, error) {
	return "Game Mode enabled", nil
}

// ExecuteDevAction runs developer-mode one-shot actions by key name
func (a *App) ExecuteDevAction(action string) (string, error) {
	runPS := func(script string) (string, error) {
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	switch action {
	case "power_ultimate":
		return a.SetPowerPlan("ultimate")
	case "power_balanced":
		return a.SetPowerPlan("balanced")
	case "defender_exclusions":
		return runPS(`Add-MpPreference -ExclusionPath "$env:USERPROFILE\source","$env:USERPROFILE\projects" -ErrorAction Stop; Write-Output "Dev folders excluded from Defender."`)
	case "defender_remove":
		return runPS(`Remove-MpPreference -ExclusionPath "$env:USERPROFILE\source","$env:USERPROFILE\projects" -ErrorAction SilentlyContinue; Write-Output "Dev folder exclusions removed."`)
	case "vscode_optimize":
		return runPS(`$s = "$env:APPDATA\Code\User\settings.json"; if (Test-Path $s) { Write-Output "VSCode settings.json found at $s" } else { Write-Output "VSCode settings.json not found." }`)
	case "vscode_highram":
		return runPS(`Start-Process "code" -ErrorAction SilentlyContinue; Write-Output "VSCode launched."`)
	case "vscode_noext":
		return runPS(`Start-Process "code" "--disable-extensions" -ErrorAction SilentlyContinue; Write-Output "VSCode launched without extensions."`)
	case "vs_multicore":
		return runPS(`Write-Output "Multi-core build tip: set /p MSBUILDNODEREUSE=0 in your build script."`)
	case "git_optimize":
		return runPS(`git config --global core.preloadindex true; git config --global core.fscache true; git config --global gc.auto 256; Write-Output "Git speed optimized."`)
	case "ping_test":
		return a.PingHost("8.8.8.8")
	case "taskmgr":
		taskmgrCmd := exec.Command("taskmgr.exe")
		taskmgrCmd.Start()
		return "Task Manager launched.", nil
	case "restart_explorer":
		return runPS(`Stop-Process -Name explorer -Force; Start-Sleep 1; Start-Process explorer; Write-Output "Explorer restarted."`)
	case "kill_chrome":
		return runPS(`Stop-Process -Name chrome -Force -ErrorAction SilentlyContinue; Write-Output "Chrome processes terminated."`)
	default:
		return fmt.Sprintf("Unknown dev action: %s", action), nil
	}
}


// GetUSBStatus returns current USB block state: "BLOCKED" or "ALLOWED"
// Reads the real registry/policy via the native audit.
func (a *App) GetUSBStatus() string {
	data, err := a.GetNativeAudit()
	if err != nil {
		return "UNKNOWN"
	}
	return data.Security.USBStatus
}

// GetRDPStatus returns current RDP block state: "BLOCKED" or "ALLOWED"
func (a *App) GetRDPStatus() string {
	data, err := a.GetNativeAudit()
	if err != nil {
		return "UNKNOWN"
	}
	return data.Security.RDPStatus
}

// GetBrowserStatus returns current Browser Policy block state: "BLOCKED" or "ALLOWED"
func (a *App) GetBrowserStatus() string {
	data, err := a.GetNativeAudit()
	if err != nil {
		return "UNKNOWN"
	}
	return data.Security.BrowserPolicy
}


// ProcessInfo holds basic process details for the Process Manager tab
type ProcessInfo struct {
	Name string `json:"Name"`
	PID  int    `json:"PID"`
	Mem  int    `json:"Mem"` // MB
}

// GetAllProcesses returns all running processes sorted by memory usage
func (a *App) GetAllProcesses() ([]ProcessInfo, error) {
	script := `
$procs = Get-Process | Where-Object {$_.Id -gt 4} | Sort-Object WorkingSet -Descending | Select-Object -First 150
$procs | ForEach-Object {
    @{ Name = $_.Name; PID = $_.Id; Mem = [math]::Round($_.WorkingSet / 1MB, 0) }
} | ConvertTo-Json -Depth 3
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var procs []ProcessInfo
	if err == nil && len(out) > 0 {
		json.Unmarshal(out, &procs)
	}
	return procs, nil
}

// KillProcess terminates a process by PID
func (a *App) KillProcess(pid int) (string, error) {
	script := fmt.Sprintf(`Stop-Process -Id %d -Force -ErrorAction Stop`, pid)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to kill PID %d: %s", pid, strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("Process %d terminated.", pid), nil
}

// BootItem holds a single startup entry's boot impact info
type BootItem struct {
	App    string `json:"App"`
	Impact string `json:"Impact"` // "Low", "Medium", "High"
	Delay  string `json:"Delay"`
}

// GetBootAnalysis returns startup apps with their boot impact from Windows
func (a *App) GetBootAnalysis() ([]BootItem, error) {
	script := `
try {
    $items = Get-CimInstance Win32_StartupCommand -ErrorAction SilentlyContinue | ForEach-Object {
        @{ App = $_.Name; Impact = "Medium"; Delay = $_.Location }
    }
    if ($null -eq $items) { $items = @() }
    @($items) | ConvertTo-Json -Depth 3
} catch { "[]" }
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var items []BootItem
	if err == nil && len(out) > 0 {
		json.Unmarshal(out, &items)
	}
	return items, nil
}

// DriverInfo holds driver health details
type DriverInfo struct {
	Name    string `json:"Name"`
	Class   string `json:"Class"`
	Version string `json:"Version"`
	Status  string `json:"Status"` // "OK", "Error", "Degraded"
}

// GetDriverHealth scans for problematic drivers using Win32_PnPSignedDriver
func (a *App) GetDriverHealth() ([]DriverInfo, error) {
	script := `
try {
    $drivers = Get-WmiObject Win32_PnPSignedDriver -ErrorAction SilentlyContinue |
        Where-Object { $_.DeviceName -ne $null -and $_.DriverVersion -ne $null } |
        Select-Object DeviceName, DeviceClass, DriverVersion,
            @{n='Status';e={ if ($_.IsSigned) {'OK'} else {'Error'} }} |
        Sort-Object Status -Descending |
        Select-Object -First 80
    if ($null -eq $drivers) { $drivers = @() }
    $drivers | ForEach-Object {
        @{ Name = $_.DeviceName; Class = $_.DeviceClass; Version = $_.DriverVersion; Status = $_.Status }
    } | ConvertTo-Json -Depth 3
} catch { "[]" }
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var drivers []DriverInfo
	if err == nil && len(out) > 0 {
		json.Unmarshal(out, &drivers)
	}
	return drivers, nil
}

// ExportSystemReport generates a plain-text system report and saves it to the Desktop
func (a *App) ExportSystemReport() (string, error) {
	data, err := a.GetNativeAudit()
	if err != nil {
		return "", fmt.Errorf("audit failed: %v", err)
	}
	ts := time.Now().Format("2006-01-02_15-04-05")
	report := fmt.Sprintf(`TGS WORLD — System Report
Generated: %s
=================================================

[SYSTEM]
  PC Name  : %s
  OS       : %s
  Build    : %s
  Domain   : %s
  User     : %s

[HARDWARE]
  CPU      : %s
  RAM      : %s
  Model    : %s

[NETWORK]
  IP       : %s
  MAC      : %s
  Gateway  : %s
  Adapter  : %s

[SECURITY]
  USB      : %s
  RDP      : %s
  Firewall : %v
  UAC      : %s
  Admin    : %v
`,
		ts,
		data.System.Name, data.System.OS, data.System.Build, data.System.Domain, data.System.User,
		data.Hardware.CPU, data.Hardware.RAMTotal, data.System.Model,
		data.Network.IP, data.Network.MAC, data.Network.Gateway, data.Network.Adapter,
		data.Security.USBStatus, data.Security.RDPStatus, data.Security.Firewall,
		data.Security.UAC, data.Security.IsAdmin,
	)

	// Save to Desktop
	desktopPath := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	outPath := filepath.Join(desktopPath, fmt.Sprintf("TGS_Report_%s.txt", ts))
	if err := os.WriteFile(outPath, []byte(report), 0644); err != nil {
		return "", fmt.Errorf("could not save report: %v", err)
	}
	return fmt.Sprintf("✅ Report saved to Desktop: TGS_Report_%s.txt", ts), nil
}

// PauseWindowsUpdate pauses Windows Update for the given number of days (0 = resume)
func (a *App) PauseWindowsUpdate(days int) (string, error) {
	var script string
	if days <= 0 {
		// Resume updates
		script = `
$wu = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU"
Remove-ItemProperty -Path $wu -Name "NoAutoUpdate" -ErrorAction SilentlyContinue
$wu2 = "HKLM:\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings"
Remove-ItemProperty -Path $wu2 -Name "PauseUpdatesExpiryTime" -ErrorAction SilentlyContinue
Remove-ItemProperty -Path $wu2 -Name "PauseFeatureUpdatesStartTime" -ErrorAction SilentlyContinue
Remove-ItemProperty -Path $wu2 -Name "PauseQualityUpdatesStartTime" -ErrorAction SilentlyContinue
Start-Service wuauserv -ErrorAction SilentlyContinue
Write-Output "Windows Update resumed."
`
	} else {
		expiry := time.Now().AddDate(0, 0, days).Format("2006-01-02T15:04:05Z")
		script = fmt.Sprintf(`
$wu2 = "HKLM:\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings"
if (-not (Test-Path $wu2)) { New-Item -Path $wu2 -Force | Out-Null }
Set-ItemProperty -Path $wu2 -Name "PauseUpdatesExpiryTime" -Value "%s" -Type String -Force
Set-ItemProperty -Path $wu2 -Name "PauseFeatureUpdatesStartTime" -Value (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") -Type String -Force
Set-ItemProperty -Path $wu2 -Name "PauseQualityUpdatesStartTime" -Value (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ") -Type String -Force
Stop-Service wuauserv -Force -ErrorAction SilentlyContinue
Write-Output "Windows Update paused for %d days."
`, expiry, days)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PauseWindowsUpdate failed: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// ScheduleAutoCleanup creates (or removes) a Windows Scheduled Task for auto cleanup
func (a *App) ScheduleAutoCleanup(intervalHours int) (string, error) {
	taskName := "TGS_AutoCleanup"
	if intervalHours <= 0 {
		// Remove task
		script := fmt.Sprintf(`schtasks /Delete /TN "%s" /F 2>&1`, taskName)
		cmd := exec.Command("cmd", "/C", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.CombinedOutput()
		return "Auto Cleanup scheduler disabled.", nil
	}
	// Create a scheduled task that runs a cleanup PowerShell inline every N hours
	cleanScript := `Clear-RecycleBin -Force -ErrorAction SilentlyContinue; ` +
		`Remove-Item "$env:TEMP\*" -Recurse -Force -ErrorAction SilentlyContinue; ` +
		`ipconfig /flushdns | Out-Null`
	psCmd := fmt.Sprintf(`powershell -NoProfile -WindowStyle Hidden -Command "%s"`, cleanScript)
	createScript := fmt.Sprintf(`schtasks /Create /SC HOURLY /MO %d /TN "%s" /TR "%s" /F /RL HIGHEST 2>&1`,
		intervalHours, taskName, psCmd)
	cmd := exec.Command("cmd", "/C", createScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ScheduleAutoCleanup failed: %s", strings.TrimSpace(string(out)))
	}
	return fmt.Sprintf("✅ Auto Cleanup scheduled every %d hours. Clears Temp files, Recycle Bin, DNS cache.", intervalHours), nil
}

