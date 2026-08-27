package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ForensicsConfig struct {
	CollectEvtx    bool `json:"collectEvtx"`
	ParseRestarts  bool `json:"parseRestarts"`
	HardwareCheck  bool `json:"hardwareCheck"`
	DockerHyperV   bool `json:"dockerHyperV"`
	GenerateHTML   bool `json:"generateHTML"`
	ZipOutput      bool `json:"zipOutput"`
}

type ForensicsResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// CollectForensics performs the forensics gathering process
func (a *App) CollectForensics(config ForensicsConfig) ForensicsResult {
	baseDir := "C:\\TGS_RCA"
	timestamp := time.Now().Format("20060102_150405")
	runDir := filepath.Join(baseDir, "Run_"+timestamp)

	dirs := []string{
		filepath.Join(runDir, "Logs"),
		filepath.Join(runDir, "Reports"),
		filepath.Join(runDir, "CSV"),
		filepath.Join(runDir, "TXT"),
	}

	a.emitProgress(5, "Creating output directories...")
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return ForensicsResult{Success: false, Message: "Failed to create directories: " + err.Error()}
		}
	}

	// 1. System Info
	a.emitProgress(10, "Collecting System Information...")
	sysInfoFile := filepath.Join(runDir, "TXT", "SystemInfo.txt")
	a.collectSystemInfo(sysInfoFile)

	// 2. Event Logs
	if config.CollectEvtx {
		a.emitProgress(30, "Exporting System and Application Event Logs...")
		a.exportEvtx(filepath.Join(runDir, "Logs", "System.evtx"), "System")
		a.exportEvtx(filepath.Join(runDir, "Logs", "Application.evtx"), "Application")
		a.exportEvtx(filepath.Join(runDir, "Logs", "Security.evtx"), "Security")
		a.exportEvtx(filepath.Join(runDir, "Logs", "WindowsUpdateClient.evtx"), "Microsoft-Windows-WindowsUpdateClient/Operational")
	}

	// 3. Parse Restart Events
	if config.ParseRestarts {
		a.emitProgress(50, "Parsing Restart & Crash Events...")
		a.parseRestartEvents(filepath.Join(runDir, "CSV", "RestartEvents.csv"))
	}

	// 4. Hardware Check
	if config.HardwareCheck {
		a.emitProgress(60, "Running Hardware Check (SMART, CPU, RAM)...")
		a.collectHardwareCheck(filepath.Join(runDir, "TXT", "Hardware.txt"))
	}

	// 5. Docker / Hyper-V
	if config.DockerHyperV {
		a.emitProgress(70, "Checking Virtualization (Docker/Hyper-V)...")
		a.collectVirtualizationInfo(filepath.Join(runDir, "TXT", "Virtualization.txt"))
	}

	// 6. Generate HTML Report
	if config.GenerateHTML {
		a.emitProgress(85, "Generating HTML Report...")
		a.generateHTMLReport(filepath.Join(runDir, "Reports", "Summary.html"), runDir)
	}

	// 7. Zip Output
	finalPath := runDir
	if config.ZipOutput {
		a.emitProgress(95, "Zipping Output Package...")
		zipPath := filepath.Join(baseDir, "TGS_Forensics_"+timestamp+".zip")
		if err := a.zipDirectory(runDir, zipPath); err == nil {
			finalPath = zipPath
			// Optional: delete runDir after zip
			// os.RemoveAll(runDir)
		}
	}

	a.emitProgress(100, "Collection Complete!")
	return ForensicsResult{
		Success: true,
		Message: "Forensics collection completed successfully.",
		Path:    finalPath,
	}
}

func (a *App) OpenOutputFolder(path string) {
	if path == "" {
		path = "C:\\TGS_RCA"
	}
	exec.Command("explorer", path).Start()
}

func (a *App) emitProgress(percent int, status string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "forensics:status", status)
		runtime.EventsEmit(a.ctx, "forensics:progress", percent)
	}
}

func (a *App) collectSystemInfo(outPath string) {
	cmd := `
	$os = Get-CimInstance Win32_OperatingSystem
	$cs = Get-CimInstance Win32_ComputerSystem
	$bios = Get-CimInstance Win32_BIOS
	$uptime = (Get-Date) - $os.LastBootUpTime

	$info = @"
Computer Name: $($cs.Name)
Username: $($cs.PrimaryOwnerName)
Domain: $($cs.Domain)
Windows Version: $($os.Caption) ($($os.Version))
Last Boot Time: $($os.LastBootUpTime)
Uptime: $($uptime.Days) Days, $($uptime.Hours) Hours, $($uptime.Minutes) Minutes
BIOS: $($bios.SMBIOSBIOSVersion) ($($bios.Manufacturer))
"@
	$info | Out-File -FilePath "` + outPath + `" -Encoding UTF8
	`
	a.runPS(cmd)
}

func (a *App) exportEvtx(outPath, logName string) {
	cmd := exec.Command("wevtutil", "epl", logName, outPath)
	cmd.Run() // Ignore errors if log doesn't exist
}

func (a *App) parseRestartEvents(outPath string) {
	cmd := `
	$events = Get-WinEvent -FilterHashtable @{LogName='System'; ID=41,6008,1074,1001,6005,6006} -MaxEvents 500 -ErrorAction SilentlyContinue
	$events | Select-Object TimeCreated, Id, LevelDisplayName, Message | Export-Csv -Path "` + outPath + `" -NoTypeInformation
	`
	a.runPS(cmd)
}

func (a *App) collectHardwareCheck(outPath string) {
	cmd := `
	$disk = Get-PhysicalDisk -ErrorAction SilentlyContinue | Select-Object DeviceId, MediaType, OperationalStatus, HealthStatus, Size
	$cpu = Get-CimInstance Win32_Processor | Select-Object Name, NumberOfCores, NumberOfLogicalProcessors
	$ram = Get-CimInstance Win32_PhysicalMemory | Select-Object DeviceLocator, Capacity, Speed

	$out = "=== DISK HEALTH ===` + "`n" + `"
	$disk | Out-String -Stream | ForEach-Object { $out += "$_` + "`n" + `" }
	
	$out += "` + "`n" + `=== CPU ===` + "`n" + `"
	$cpu | Out-String -Stream | ForEach-Object { $out += "$_` + "`n" + `" }
	
	$out += "` + "`n" + `=== RAM ===` + "`n" + `"
	$ram | Out-String -Stream | ForEach-Object { $out += "$_` + "`n" + `" }

	$out | Out-File -FilePath "` + outPath + `" -Encoding UTF8
	`
	a.runPS(cmd)
}

func (a *App) collectVirtualizationInfo(outPath string) {
	cmd := `
	$out = "=== DOCKER ===` + "`n" + `"
	if (Get-Command docker -ErrorAction SilentlyContinue) {
		$out += (docker info 2>&1 | Out-String) + "` + "`n" + `"
		$out += (docker ps 2>&1 | Out-String) + "` + "`n" + `"
	} else {
		$out += "Docker not installed or not in PATH.` + "`n" + `"
	}

	$out += "` + "`n" + `=== HYPER-V ===` + "`n" + `"
	if (Get-Command Get-VM -ErrorAction SilentlyContinue) {
		$out += (Get-VM | Out-String 2>&1) + "` + "`n" + `"
	} else {
		$out += "Hyper-V module not available.` + "`n" + `"
	}

	$out | Out-File -FilePath "` + outPath + `" -Encoding UTF8
	`
	a.runPS(cmd)
}

func (a *App) generateHTMLReport(outPath, runDir string) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>TGS Forensic Report</title>
	<style>
		body { font-family: Arial, sans-serif; background-color: #1e1e1e; color: #fff; margin: 20px; }
		h1 { color: #00bcd4; }
		h2 { color: #ff9800; border-bottom: 1px solid #444; padding-bottom: 5px; }
		pre { background-color: #2d2d2d; padding: 10px; border-radius: 5px; overflow-x: auto; }
	</style>
</head>
<body>
	<h1>TGS Windows Server Forensic Report</h1>
	<p>Generated on: ` + time.Now().Format(time.RFC1123) + `</p>
	
	<h2>System Information</h2>
	<pre>` + a.readFile(filepath.Join(runDir, "TXT", "SystemInfo.txt")) + `</pre>

	<h2>Hardware Status</h2>
	<pre>` + a.readFile(filepath.Join(runDir, "TXT", "Hardware.txt")) + `</pre>

	<h2>Virtualization</h2>
	<pre>` + a.readFile(filepath.Join(runDir, "TXT", "Virtualization.txt")) + `</pre>

	<h2>Recent Restart Events (See CSV for full list)</h2>
	<p>Check the RestartEvents.csv in the CSV folder for a complete timeline.</p>
</body>
</html>`
	os.WriteFile(outPath, []byte(html), 0644)
}

func (a *App) readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "File not found or error reading: " + err.Error()
	}
	return string(b)
}

func (a *App) runPS(script string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func (a *App) zipDirectory(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Create header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		
		// Handle path for zip file structure
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		
		header.Name = filepath.ToSlash(relPath)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
	return nil
}

type RCAResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	Cause      string `json:"cause"`
	Evidence   string `json:"evidence"`
	Confidence string `json:"confidence"`
	Details    string `json:"details"`
}

// SelectEvtxFile opens a native file dialog to pick an .evtx file
func (a *App) SelectEvtxFile() string {
	if a.ctx == nil {
		return ""
	}
	path, _ := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Offline Event Log",
		Filters: []runtime.FileFilter{
			{DisplayName: "Windows Event Log (*.evtx)", Pattern: "*.evtx"},
		},
	})
	return path
}

// AnalyzeOfflineEvtx parses a given .evtx file and runs RCA
func (a *App) AnalyzeOfflineEvtx(filePath string) RCAResult {
	if filePath == "" || filepath.Ext(filePath) != ".evtx" {
		return RCAResult{Success: false, Message: "Invalid file selected."}
	}

	a.emitProgress(10, "Parsing offline .evtx file...")
	
	// We'll extract 1074, 1001, 6008, 41 directly from the offline file
	cmd := fmt.Sprintf(`
	$events = Get-WinEvent -FilterHashtable @{Path='%s'; ID=41,6008,1074,1001} -MaxEvents 50 -ErrorAction SilentlyContinue
	$events | ForEach-Object {
		$_.Id.ToString() + "|" + $_.TimeCreated.ToString("yyyy-MM-dd HH:mm:ss") + "|" + $_.Message.Replace("` + "`n" + `", " ").Replace("` + "`r" + `", "")
	}
	`, filePath)
	
	out := a.runPS(cmd)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	
	if len(lines) == 0 || lines[0] == "" {
		return RCAResult{
			Success: true,
			Message: "Analysis complete. No critical restart events found.",
			Cause: "Normal Operation / Unknown",
			Confidence: "Low",
			Evidence: "No relevant IDs (41, 1001, 1074, 6008) found in top 50 events.",
		}
	}

	a.emitProgress(60, "Running RCA Engine on parsed events...")

	// Simple RCA logic: Look at the most recent critical event (first line)
	firstEvent := strings.SplitN(strings.TrimSpace(lines[0]), "|", 3)
	if len(firstEvent) < 3 {
		return RCAResult{Success: false, Message: "Failed to parse event data."}
	}

	id := firstEvent[0]
	timeStr := firstEvent[1]
	msg := firstEvent[2]

	res := RCAResult{
		Success:    true,
		Message:    "Analysis complete.",
		Details:    fmt.Sprintf("Most recent event at %s: ID %s - %s", timeStr, id, msg),
	}

	switch id {
	case "1074":
		if strings.Contains(strings.ToLower(msg), "windowsupdate") || strings.Contains(strings.ToLower(msg), "wusa.exe") {
			res.Cause = "Windows Update Reboot"
			res.Evidence = "Event 1074 initiated by WindowsUpdateClient / TrustedInstaller"
			res.Confidence = "98%"
		} else if strings.Contains(strings.ToLower(msg), "svchost.exe") || strings.Contains(strings.ToLower(msg), "explorer.exe") {
			res.Cause = "User Initiated Restart"
			res.Evidence = "Event 1074 initiated by svchost / explorer"
			res.Confidence = "85%"
		} else {
			res.Cause = "Planned Restart (Other)"
			res.Evidence = "Event 1074 found without explicit Update signature"
			res.Confidence = "60%"
		}
	case "6008":
		res.Cause = "Unexpected Shutdown (Power loss / Hard crash)"
		res.Evidence = "Event 6008 detected indicating the previous shutdown was unexpected."
		res.Confidence = "95%"
	case "41":
		res.Cause = "Kernel-Power Failure"
		res.Evidence = "Event 41 (Kernel-Power) means the system rebooted without cleanly shutting down first."
		res.Confidence = "90%"
	case "1001":
		res.Cause = "BugCheck (BSOD)"
		res.Evidence = "Event 1001 means the computer rebooted from a bugcheck."
		res.Confidence = "99%"
	default:
		res.Cause = "Unknown"
		res.Evidence = fmt.Sprintf("Event %s logged.", id)
		res.Confidence = "N/A"
	}

	a.emitProgress(100, "RCA Complete.")
	
	// Cache the result for the Dashboard
	GlobalRCAResult = &res

	return res
}

// GlobalRCAResult stores the latest RCA result for Dashboard health checks
var GlobalRCAResult *RCAResult

// GetLastRCAResult returns the most recent RCA analysis
func (a *App) GetLastRCAResult() *RCAResult {
	return GlobalRCAResult
}

// GenerateAIPrompt formats a prompt for the user to copy/paste into ChatGPT
func (a *App) GenerateAIPrompt(rca RCAResult) string {
	prompt := fmt.Sprintf(`You are an expert Windows Server Forensic Analyst. 
I have extracted the following critical event from a server that recently experienced an issue.

Event Details:
- Identified Cause: %s
- Confidence: %s
- Evidence: %s
- Raw Log: %s

Please provide:
1. A brief explanation of what this event means in a Windows environment.
2. The top 3 troubleshooting steps I should take immediately to resolve or prevent this in the future.
3. Any specific KB articles or known Windows bugs related to this behavior.`, rca.Cause, rca.Confidence, rca.Evidence, rca.Details)
	return prompt
}

// SmtpConfig holds email credentials
type SmtpConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	To       string `json:"to"`
}

// EmailReport sends the zip file or text report via email using PowerShell
func (a *App) EmailReport(config SmtpConfig, attachmentPath string) string {
	if config.Server == "" || config.To == "" {
		return "Invalid SMTP configuration."
	}
	
	// Escape path
	attachmentPath = strings.ReplaceAll(attachmentPath, "'", "''")

	script := fmt.Sprintf(`
$SMTPClient = New-Object Net.Mail.SmtpClient('%s', %d)
$SMTPClient.EnableSsl = $true
$SMTPClient.Credentials = New-Object System.Net.NetworkCredential('%s', '%s')
$Msg = New-Object Net.Mail.MailMessage
$Msg.From = '%s'
$Msg.To.Add('%s')
$Msg.Subject = 'TGS Forensic Report'
$Msg.Body = 'Please find the attached TGS Forensic Report and RCA analysis.'
`, config.Server, config.Port, config.Username, config.Password, config.Username, config.To)

	if attachmentPath != "" {
		script += fmt.Sprintf(`
if (Test-Path '%s') {
	$Attachment = New-Object Net.Mail.Attachment('%s')
	$Msg.Attachments.Add($Attachment)
}
`, attachmentPath, attachmentPath)
	}

	script += `
try {
	$SMTPClient.Send($Msg)
	Write-Output "SUCCESS"
} catch {
	Write-Output "ERROR: $($_.Exception.Message)"
} finally {
	if ($Attachment) { $Attachment.Dispose() }
	$Msg.Dispose()
}
`
	out := a.runPS(script)
	if strings.Contains(out, "SUCCESS") {
		return "Email sent successfully!"
	}
	return "Failed to send email: " + out
}
