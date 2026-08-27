# TGS All-In-One — Version Changelog

> **Author:** Yadav Guru Prasad  
> **Organization:** Triveni Technology Solutions (TGS World)  
> **Project:** TGS All-In-One IT Management Tool

---

## ✅ V14.1 — Final Release *(Current)*
> **File:** `TGS_All_In_One_v14_1_Themes_Final.exe`  
> **Date:** March 2026

### 🆕 New Features
- Futuristic sidebar UI with gradient active tab highlight
- Detailed **Dashboard** with 5 live stat cards (CPU, RAM, Virtual Memory, Storage, Network) + 4 detail tabs (Specs / CPU & Power / Memory / Network)
- **Smart Installer** — scan-first detection flow, per-app ✅/❌ status badges, Install / Reinstall / Remove buttons
- TGS Console with export, clear, Max/Min controls
- Futuristic glassmorphism card design with purple/cyan neon accents
- Google Fonts (Inter, Poppins) for improved typography
- Status indicator (Online/Offline) in sidebar footer
- Version badge `V14.1 · Active` displayed in sidebar

### 🔧 Improvements
- Dashboard now shows DIMM slot info, PageFile, Power Plan, Game Mode, GPU Scheduling
- Version string fixed to display `V14.1` correctly at all times
- Cleaner, Optimizer, Debloater tabs retained from original
- All new component files cleaned up (no extra/unused tabs in sidebar)
- Removed old `.exe` backups from `extra/` and `build/bin/`

### 🐛 Fixed
- Duplicate `build\bin\build\bin` folder removed
- `RunPerformanceTweak` missing backend reference fixed in Dashboard
- App reset to 7-tab layout (no experimental tabs in sidebar)

---

## V14.1 — Beta / Theme Build
> **File:** `TGS_All_In_One_v14_1_Themes_Final.exe` *(early beta)*  
> **Date:** March 2026 (early)

- Added multi-theme support (Deep Space, Clean, Solarized, Hacker, Oceanic)
- First implementation of futuristic header with search bar (Ctrl+K)
- Introduced glassmorphism card styling
- Added TGS Console terminal panel
- Experimental tabs: SystemSetup, NetworkAudit, SecurityCenter, Logs, Settings

---

## V14.1 — Initial
> **File:** `TGS_All_In_One_v14_1.exe`

- Performance Tweaks (Tweaks/Optimizer tab) added
- Startup Manager integration
- Advanced Cleaner (browser cache, Windows logs, junk files)
- Chris Titus WinUtil integration (Standard & Minimal modes)
- Restore Settings button for all tweaks

---

## V14 — Stable
> **File:** `TGS_All_In_One_v14.exe`

- Debloater tab — UWP app removal, registry tweaks, Default/Custom setup wizard
- Safety features: System Restore Point creation before debloat
- Stealth mode — hidden PowerShell windows during silent script execution
- Full Audit & Security tab with data upload to Google Sheets
- Scheduled silent audit (background reporting)
- Real-time system stats: CPU, RAM, Disk, Network

---

## V13 — Foundation
> **File:** `TGS_All_In_One_v13.exe` *(archived)*

- First Wails (Go + Svelte) implementation
- Basic Dashboard, Setup, Installer, Audit tabs
- PowerShell-based software installation
- Network config (static IP, DNS)
- Basic security toggles (USB, RDP, Browser Policy)

---

## 📋 Version Summary Table

| Version | File | Status | Key Feature |
|---------|------|--------|-------------|
| **V19.3 Final** | `...V19.3_Final.exe` | ✅ **CURRENT** | Smart Installer + Detailed Dashboard |
| V14.1 Beta | `...v14_1_Themes_Final.exe` | 🗂️ Archived | Themes + Glassmorphism |
| V14.1 Initial | `...v14_1.exe` | 🗂️ Archived | Tweaks + Debloater |
| V14 Stable | `...v14.exe` | 🗂️ Archived | Full Feature Set |
| V13 | `...v13.exe` | 🗂️ Archived | First Wails Build |

---

## 🚀 How to Release a New Version

1. Make code changes
2. Open terminal in project root
3. Run:
   ```powershell
   wails build -o .\build\bin\TGS_All_In_One_v15.exe
   ```
4. Update this `CHANGELOG.md`
5. Commit in GitHub Desktop with message: `v15 - [brief description]`
6. Push to GitHub
7. Create a new Release on GitHub with the `.exe` attached
