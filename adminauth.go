package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ─────────────────────────────────────────────────────────────────
// TGS ADMIN PASSWORD — secure tamper-prevention gate
// Password is stored as SHA-256 hash in C:\ProgramData\TGS_Monitor\admin.key
// Default password: TGS@Admin (must be changed on first use)
// ─────────────────────────────────────────────────────────────────

const (
	adminKeyFile    = `C:\ProgramData\TGS_Monitor\admin.key`
	defaultAdminPwd = "TGS@Admin"
)

// hashPassword returns the SHA-256 hex of a password string
func hashPassword(pwd string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(pwd)))
	return fmt.Sprintf("%x", h)
}

// ensureAdminKeyFile makes sure the key file and directory exist.
// If no password has been set yet, creates the default.
func ensureAdminKeyFile() {
	dir := filepath.Dir(adminKeyFile)
	os.MkdirAll(dir, 0755)
	if _, err := os.Stat(adminKeyFile); os.IsNotExist(err) {
		// Write default password hash
		os.WriteFile(adminKeyFile, []byte(hashPassword(defaultAdminPwd)), 0600)
	}
}

// storedHash returns the hash currently stored in the key file
func storedHash() string {
	ensureAdminKeyFile()
	b, err := os.ReadFile(adminKeyFile)
	if err != nil {
		return hashPassword(defaultAdminPwd) // fallback
	}
	return strings.TrimSpace(string(b))
}

// ── Exposed to Wails frontend ──────────────────────────────────────

// VerifyAdminPassword returns true if the given password matches the stored hash.
func (a *App) VerifyAdminPassword(password string) bool {
	if strings.TrimSpace(password) == "" {
		return false
	}
	return hashPassword(password) == storedHash()
}

// HasAdminPassword returns true once a password file exists
// (so the UI knows whether this is first-time setup or not).
func (a *App) HasAdminPassword() bool {
	_, err := os.Stat(adminKeyFile)
	return err == nil
}

// SetAdminPassword changes the admin password.
// currentPwd must match the existing hash (or be the default).
// Returns a success message or an error.
func (a *App) SetAdminPassword(currentPwd, newPwd string) (string, error) {
	newPwd = strings.TrimSpace(newPwd)
	if newPwd == "" {
		return "", fmt.Errorf("new password cannot be empty")
	}
	if len(newPwd) < 6 {
		return "", fmt.Errorf("password must be at least 6 characters")
	}

	// Verify current password
	if !a.VerifyAdminPassword(currentPwd) {
		return "", fmt.Errorf("current password is incorrect")
	}

	ensureAdminKeyFile()
	if err := os.WriteFile(adminKeyFile, []byte(hashPassword(newPwd)), 0600); err != nil {
		return "", fmt.Errorf("failed to save new password: %v", err)
	}
	return "✅ Admin password updated successfully.", nil
}

// ResetAdminPassword resets the password back to default without needing the old one.
// This should only be called from a trusted context (e.g. running as SYSTEM).
func (a *App) ResetAdminPassword() (string, error) {
	ensureAdminKeyFile()
	if err := os.WriteFile(adminKeyFile, []byte(hashPassword(defaultAdminPwd)), 0600); err != nil {
		return "", fmt.Errorf("reset failed: %v", err)
	}
	return fmt.Sprintf("✅ Password reset to default: %s", defaultAdminPwd), nil
}
