package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)


// ─────────────────────────────────────────────────────────────────
// TGS WORLD SECURITY SERVICE  (real Windows Service)
// Called from main.go when the -svc flag is detected.
// ─────────────────────────────────────────────────────────────────

const (
	svcName    = "TGS WORLD Security Monitor"
	svcDisplay = "TGS WORLD Security Monitor"
	svcDesc    = "TGS WORLD – Continuous RDP and browser security monitoring with email alerts."
	configDir  = `C:\ProgramData\TGS_Monitor`
	configFile = `C:\ProgramData\TGS_Monitor\monitor.cfg`
	logFile    = `C:\ProgramData\TGS_Monitor\monitor.log`
	pendFile   = `C:\ProgramData\TGS_Monitor\pending.txt`
)

// ──── entry point called from main.go ─────────────────────────────
func RunAsService() {
	svc.Run(svcName, &tgsService{})
}

// ──── Windows Service handler ─────────────────────────────────────
type tgsService struct{}

func (m *tgsService) Execute(_ []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	stop := make(chan struct{})
	go svcMonitorLoop(stop)

	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			status <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			status <- svc.Status{State: svc.StopPending}
			close(stop)
			return false, 0
		}
	}
}

// ──── Main monitoring loop ─────────────────────────────────────────
func svcMonitorLoop(stop <-chan struct{}) {
	lastAlerts := map[string]interface{}{}

	appendLog := func(msg string) {
		ts := time.Now().Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("[%s] %s\n", ts, msg)
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(line)
			f.Close()
		}
	}

	for {
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}

		cfg := readConfig()
		alerts := []string{}

		// ── 1. RDP CLIPBOARD TAMPER ────────────────────────────
		rdpPolicyPath := filepath.Join(configDir, "rdp_policy.txt")
		rdpIntentionallyAllowed := false
		if policyBytes, pErr := os.ReadFile(rdpPolicyPath); pErr == nil {
			rdpIntentionallyAllowed = strings.TrimSpace(string(policyBytes)) == "ALLOWED"
		}
		if !rdpIntentionallyAllowed {
			rk, err := registry.OpenKey(registry.LOCAL_MACHINE,
				`SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services`,
				registry.QUERY_VALUE|registry.SET_VALUE)
			if err == nil {
				val, _, rerr := rk.GetIntegerValue("fDisableClip")
				if rerr != nil || val != 1 {
					alerts = append(alerts, "[RDP BREACH] RDP Clipboard block removed – re-enforcing")
					rk.SetDWordValue("fDisableClip", 1)
				}
				rk.Close()
			}
		}

		// ── 2. BROWSER HISTORY – Gmail unauthorized login ─────
		gmailAlerts := scanBrowserHistory(cfg, lastAlerts)
		alerts = append(alerts, gmailAlerts...)

		// ── 5. SEND ALERTS ─────────────────────────────────────
		if len(alerts) > 0 {
			ts := time.Now().Format("2006-01-02 15:04:05")
			for _, a := range alerts {
				appendLog(a)
			}
			showToastViaHelper(alerts)

			body := buildEmailBody(alerts, ts)
			subject := fmt.Sprintf("[TGS WORLD] Security Alert on %s — %s", hostname(), ts)

			if hasInternet() {
				if err := sendMail(cfg, subject, body); err != nil {
					appendLog("Email failed, buffering: " + err.Error())
					bufferAlert(subject, body)
				} else {
					appendLog("Email sent OK")
				}
			} else {
				appendLog("No internet – alert buffered")
				bufferAlert(subject, body)
			}
		}

		// ── 6. FLUSH BUFFERED ALERTS if internet back ─────────
		if hasInternet() {
			flushPending(cfg, appendLog)
		}
	}
}

// ──── Config reader ────────────────────────────────────────────────
type monitorCfg struct {
	AlertEmails    []string
	SMTPServer     string
	SMTPUser       string
	SMTPPass       string
	AllowedDomains []string
	FromName       string
}

func readConfig() monitorCfg {
	var c monitorCfg
	c.SMTPServer = "smtp.gmail.com"
	c.FromName = "TGS WORLD Security"
	data, err := os.ReadFile(configFile)
	if err != nil {
		return c
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch k {
		case "AlertEmail":
			for _, e := range strings.Split(v, ",") {
				if t := strings.TrimSpace(e); t != "" {
					c.AlertEmails = append(c.AlertEmails, t)
				}
			}
		case "SMTPServer":
			if v != "" {
				c.SMTPServer = v
			}
		case "SMTPUser":
			c.SMTPUser = v
		case "SMTPPass":
			c.SMTPPass = v
		case "AllowedDomains":
			for _, d := range strings.Split(v, ",") {
				clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(d), "*@"))
				if clean != "" {
					c.AllowedDomains = append(c.AllowedDomains, clean)
				}
			}
		case "FromName":
			if v != "" {
				c.FromName = v
			}
		}
	}
	return c
}

// ──── Browser history scanner ──────────────────────────────────────
var histLastTimes = map[string]time.Time{}
var histLastCounts = map[string]int{}

func scanBrowserHistory(cfg monitorCfg, lastAlerts map[string]interface{}) []string {
	appData := os.Getenv("APPDATA")
	localApp := os.Getenv("LOCALAPPDATA")
	patterns := []string{
		filepath.Join(localApp, `Google\Chrome\User Data\*\History`),
		filepath.Join(localApp, `Microsoft\Edge\User Data\*\History`),
		filepath.Join(appData, `Opera Software\Opera Stable\History`),
		filepath.Join(appData, `Opera Software\Opera GX Stable\History`),
		filepath.Join(localApp, `BraveSoftware\Brave-Browser\User Data\*\History`),
	}

	var alerts []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, histFile := range matches {
			info, err := os.Stat(histFile)
			if err != nil {
				continue
			}
			modTime := info.ModTime()
			prevTime, seen := histLastTimes[histFile]
			if seen && !modTime.After(prevTime) {
				continue
			}
			histLastTimes[histFile] = modTime

			// Read raw bytes (SQLite stores URLs as UTF-8 text)
			data, err := readFileCopy(histFile)
			if err != nil {
				continue
			}
			text := string(data)

			// Find google accounts URLs and extract hinted email
			urlPattern := "accounts.google.com"
			newCounts := map[string]int{}
			idx := 0
			for {
				pos := strings.Index(text[idx:], urlPattern)
				if pos < 0 {
					break
				}
				abs := idx + pos
				idx = abs + 1
				// Extract up to 400 chars of URL context
				end := abs + 400
				if end > len(text) {
					end = len(text)
				}
				chunk := text[abs:end]
				hint := extractEmailHint(chunk)
				if hint == "" {
					continue
				}
				allowed := false
				for _, dom := range cfg.AllowedDomains {
					if dom != "" && strings.Contains(hint, dom) {
						allowed = true
						break
					}
				}
				if !allowed {
					newCounts[hint]++
				}
			}

			if seen {
				for hint, count := range newCounts {
					trackKey := histFile + "_" + hint
					if count > histLastCounts[trackKey] {
						alerts = append(alerts, "[GMAIL] Unauthorized login attempt: "+hint)
					}
				}
			}
			for hint, count := range newCounts {
				histLastCounts[histFile+"_"+hint] = count
			}
		}
	}
	return alerts
}

func extractEmailHint(chunk string) string {
	for _, param := range []string{"Email=", "hd="} {
		idx := strings.Index(chunk, param)
		if idx >= 0 {
			val := chunk[idx+len(param):]
			end := strings.IndexAny(val, "& \x00\r\n")
			if end > 0 {
				return val[:end]
			}
			if len(val) > 80 {
				return val[:80]
			}
			return val
		}
	}
	return ""
}

func readFileCopy(src string) ([]byte, error) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("tgs_hist_%d.db", time.Now().UnixNano()))
	defer os.Remove(tmp)
	
	// esentutl /y copies locked files (like open SQLite databases) by bypassing locks
	cmd := execHidden("esentutl.exe", "/y", src, "/d", tmp, "/o")
	if err := cmd.Run(); err != nil {
		// Fallback to standard copy if esentutl fails
		in, err := os.Open(src)
		if err != nil {
			return nil, err
		}
		defer in.Close()
		out, err := os.Create(tmp)
		if err != nil {
			return nil, err
		}
		io.Copy(out, in)
		out.Close()
	}
	
	return os.ReadFile(tmp)
}

// ──── Email with STARTTLS ──────────────────────────────────────────
func sendMail(cfg monitorCfg, subject, body string) error {
	if len(cfg.AlertEmails) == 0 || cfg.SMTPUser == "" || cfg.SMTPPass == "" {
		return fmt.Errorf("email not configured")
	}
	from := fmt.Sprintf("%s <%s>", cfg.FromName, cfg.SMTPUser)
	to := strings.Join(cfg.AlertEmails, ", ")
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, to, subject, body)

	host := cfg.SMTPServer
	addr := net.JoinHostPort(host, "587")
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return err
	}
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, host)
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(cfg.SMTPUser); err != nil {
		return err
	}
	for _, r := range cfg.AlertEmails {
		c.Rcpt(strings.TrimSpace(r))
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	w.Write([]byte(msg))
	w.Close()
	return c.Quit()
}

// ──── Offline buffer ───────────────────────────────────────────────
func bufferAlert(subject, body string) {
	safe := strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "{NL}"), "\n", "{NL}")
	line := fmt.Sprintf("%s|||%s|||%s\n", time.Now().Format("2006-01-02 15:04:05"), subject, safe)
	f, _ := os.OpenFile(pendFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.WriteString(line)
		f.Close()
	}
}

func flushPending(cfg monitorCfg, logFn func(string)) {
	data, err := os.ReadFile(pendFile)
	if err != nil || len(data) == 0 {
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	allOK := true
	for _, line := range lines {
		parts := strings.SplitN(line, "|||", 3)
		if len(parts) < 3 {
			continue
		}
		subj := parts[1]
		body := strings.ReplaceAll(parts[2], "{NL}", "\n")
		if err := sendMail(cfg, subj, body); err != nil {
			allOK = false
			logFn("Flush failed: " + err.Error())
		}
	}
	if allOK {
		os.Remove(pendFile)
		logFn("All buffered alerts sent successfully")
	}
}

// ──── Internet check ───────────────────────────────────────────────
func hasInternet() bool {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ──── Toast notification from service (Session 0 workaround) ──────
// Writes a VBScript that uses WScript.Shell to show a toast from the user session.
func showToastViaHelper(alerts []string) {
	msg := strings.Join(alerts, "; ")
	if len(msg) > 250 {
		msg = msg[:250]
	}
	// Escape for VBScript double-quoted string
	msg = strings.ReplaceAll(msg, `"`, `'`)
	vbs := fmt.Sprintf(`
Set shell = CreateObject("WScript.Shell")
shell.Run "powershell -WindowStyle Hidden -Command ""[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null; $xml = '<toast><visual><binding template=""ToastText02""><text id=""1"">TGS WORLD Security Alert</text><text id=""2"">%s</text></binding></visual></toast>'; $d = [Windows.Data.Xml.Dom.XmlDocument]::new(); $d.LoadXml($xml); [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('TGS WORLD').Show([Windows.UI.Notifications.ToastNotification]::new($d))""", 0, False
`, msg)
	vbsPath := filepath.Join(configDir, "toast_helper.vbs")
	os.WriteFile(vbsPath, []byte(vbs), 0644)
	execHidden("wscript.exe", "//B", "//Nologo", vbsPath).Start()
}

// ──── Helpers ──────────────────────────────────────────────────────
func hostname() string {
	h, _ := os.Hostname()
	return h
}

func buildEmailBody(alerts []string, ts string) string {
	h, _ := os.Hostname()
	return fmt.Sprintf("TGS WORLD Security Alert\n\nEvents:\n%s\n\nTime: %s\nPC Name: %s\n",
		strings.Join(alerts, "\n"), ts, h)
}

// ─────────────────────────────────────────────────────────────────
// SERVICE INSTALL / UNINSTALL   (called from native.go)
// ─────────────────────────────────────────────────────────────────

// InstallWindowsService copies this EXE and registers it as a real Windows Service.
func (a *App) InstallWindowsService(alertEmail, smtpServer, smtpUser, smtpPass, allowedDomains, fromName string) (string, error) {
	os.MkdirAll(configDir, 0755)

	// ── Write config ────────────────────────────────────────────────────────
	if smtpServer == "" { smtpServer = "smtp.gmail.com" }
	if fromName == ""   { fromName = "TGS WORLD Security" }
	cfg := fmt.Sprintf("AlertEmail=%s\nSMTPServer=%s\nSMTPUser=%s\nSMTPPass=%s\nAllowedDomains=%s\nFromName=%s\n",
		alertEmail, smtpServer, smtpUser, smtpPass, allowedDomains, fromName)
	if err := os.WriteFile(configFile, []byte(cfg), 0600); err != nil {
		return "", fmt.Errorf("config write failed: %w", err)
	}

	// ── STEP 1: Kill everything related to old service ──────────────────────
	exec.Command("schtasks", "/Delete", "/TN", "TGS WORLD Security Monitor", "/F").Run()
	exec.Command("schtasks", "/Delete", "/TN", "TGS_Login_Monitor", "/F").Run()
	execHidden("sc.exe", "stop", svcName).Run()
	time.Sleep(2 * time.Second)
	execHidden("taskkill", "/F", "/IM", "TGSWorldSvc.exe").Run()
	time.Sleep(500 * time.Millisecond)

	// ── STEP 2: Delete via sc.exe (handles SCM timing internally) ──────────
	execHidden("sc.exe", "delete", svcName).Run()

	// ── STEP 3: Poll REGISTRY until service key is fully gone ──────────────
	// The Go mgr API has race conditions; reading the registry is definitive.
	svcRegKey := `SYSTEM\CurrentControlSet\Services\` + svcName
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, svcRegKey, registry.QUERY_VALUE)
		if err != nil {
			break // key is gone — safe to proceed
		}
		k.Close()
	}
	time.Sleep(500 * time.Millisecond) // one final breath

	// ── STEP 4: Copy EXE (file is now free) ────────────────────────────────
	exePath, _ := os.Executable()
	svcExe := filepath.Join(configDir, "TGSWorldSvc.exe")
	if err := copyFile(exePath, svcExe); err != nil {
		return "", fmt.Errorf("EXE copy failed: %w", err)
	}

	// ── STEP 5: Create service via sc.exe (no Go API, no race condition) ───
	// sc.exe requires a space after each = sign — these are separate args.
	binPath := fmt.Sprintf("%s -svc", svcExe)
	out, err := exec.Command("cmd", "/C",
		fmt.Sprintf(`sc create "%s" binPath= "%s" start= auto obj= LocalSystem DisplayName= "%s"`,
			svcName, binPath, svcDisplay)).CombinedOutput()
	if err != nil || strings.Contains(strings.ToLower(string(out)), "failed") {
		return "", fmt.Errorf("sc create failed: %s", strings.TrimSpace(string(out)))
	}

	// ── STEP 6: Description, recovery & protection via sc.exe ─────────────
	exec.Command("cmd", "/C",
		fmt.Sprintf(`sc description "%s" "%s"`, svcName, svcDesc)).Run()

	exec.Command("cmd", "/C",
		fmt.Sprintf(`sc failure "%s" reset= 86400 actions= restart/2000/restart/5000/restart/10000`,
			svcName)).Run()

	// ── STEP 7: Start the service ──────────────────────────────────────────
	startOut, startErr := exec.Command("cmd", "/C",
		fmt.Sprintf(`sc start "%s"`, svcName)).CombinedOutput()
	if startErr != nil && !strings.Contains(string(startOut), "1056") {
		// 1056 = service already running — that is fine
		return "", fmt.Errorf("service created OK but start failed: %s", strings.TrimSpace(string(startOut)))
	}

	// ── STEP 8: Protect DACL — non-Admins can view but NOT stop/delete ────
	protectSddl := `D:(A;;CCLCSWLOCRRC;;;AU)(A;;CCLCSWRPWPDTLOCRSDRCWDWO;;;BA)(A;;CCLCSWRPWPDTLOCRSDRCWDWO;;;SY)`
	exec.Command("cmd", "/C",
		fmt.Sprintf(`sc sdset "%s" "%s"`, svcName, protectSddl)).Run()

	return fmt.Sprintf("✅ TGS WORLD Security Monitor installed & running. Visible in services.msc. Alerts → %s", alertEmail), nil
}


// UninstallWindowsService stops and removes the Windows Service.
func (a *App) UninstallWindowsService() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("cannot connect to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(svcName)
	if err != nil {
		// Also clean up scheduled tasks (legacy)
		exec.Command("schtasks", "/Delete", "/TN", "TGS WORLD Security Monitor", "/F").Run()
		exec.Command("schtasks", "/Delete", "/TN", "TGS_Login_Monitor", "/F").Run()
		return "Service was not installed (already removed).", nil
	}
	defer s.Close()

	s.Control(svc.Stop)
	time.Sleep(2 * time.Second)
	s.Delete()

	// Also clean up scheduled tasks (legacy)
	exec.Command("schtasks", "/Delete", "/TN", "TGS WORLD Security Monitor", "/F").Run()
	exec.Command("schtasks", "/Delete", "/TN", "TGS_Login_Monitor", "/F").Run()

	return "TGS WORLD Security Monitor service stopped and removed.", nil
}

// GetWindowsServiceStatus returns ACTIVE or INACTIVE.
func (a *App) GetWindowsServiceStatus() string {
	m, err := mgr.Connect()
	if err != nil {
		return "INACTIVE"
	}
	defer m.Disconnect()
	s, err := m.OpenService(svcName)
	if err != nil {
		return "INACTIVE"
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return "INACTIVE"
	}
	if st.State == svc.Running {
		return "ACTIVE"
	}
	return "INACTIVE"
}

// ──── File copy helper ──────────────────────────────────────────────
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ──── execHidden: run a process with no console window ────────────
func execHidden(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

// ──── Suppress unused import warning (windows package) ──────────────
var _ = windows.ERROR_SUCCESS
