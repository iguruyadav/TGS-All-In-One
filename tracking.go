package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
)

// GetProductKeys extracts the Windows product key using WMI/PowerShell
func (a *App) GetProductKeys() (string, error) {
	script := `(Get-WmiObject -query 'select * from SoftwareLicensingService').OA3xOriginalProductKey`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		// Try fallback registry path
		fallbackScript := `(Get-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\SoftwareProtectionPlatform').BackupProductKeyDefault`
		cmdFallback := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", fallbackScript)
		cmdFallback.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		outFallback, _ := cmdFallback.CombinedOutput()
		key = strings.TrimSpace(string(outFallback))
		if key == "" {
			key = "Digital License / Not Found"
		}
	}
	return key, nil
}

// SendWebhookAlert sends a JSON payload to a Discord/Slack webhook
func (a *App) SendWebhookAlert(url, message string) (string, error) {
	if url == "" {
		return "Webhook URL is empty", nil
	}
	payload := map[string]string{"content": message}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return "Webhook sent successfully", nil
}

// GetDetailedSystemLogs gets extended Windows Event Logs (last 14 days, up to 100 logs)
func (a *App) GetDetailedSystemLogs() ([]HealthLog, error) {
	script := `
    $evts = Get-WinEvent -FilterHashtable @{LogName='System','Application'; Level=1,2; StartTime=(Get-Date).AddDays(-14)} -MaxEvents 100 -ErrorAction SilentlyContinue
    $logs = $evts | ForEach-Object {
        $msg = ""
        if ($_.Message) {
            $msg = ($_.Message -replace '\r\n', ' ' -replace '\n', ' ').Trim()
            if ($msg.Length -gt 300) { $msg = $msg.Substring(0, 300) + "..." }
        }
        @{
            Time = $_.TimeCreated.ToString("yyyy-MM-dd HH:mm")
            Level = if ($_.Level -eq 1) { "Critical" } else { "Error" }
            Source = $_.ProviderName
            EventID = [int]$_.Id
            Message = $msg
        }
    }
    @($logs) | ConvertTo-Json -Depth 5
    `
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	var logs []HealthLog
	if err == nil {
		json.Unmarshal(out, &logs)
		logs = enrichLogs(logs)
	}
	return logs, nil
}
