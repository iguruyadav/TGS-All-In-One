<#
 =====================================================
 TGS WORLD – SYSTEM INITIALIZATION (V10.7 FIXED BUTTONS)
 Updates:
 - RESTORED: Exact V10.5 Code (Stable, Fixed Layout).
 - FIXED ONLY: Button widths and positions in Software Tab.
 - NO experimental auto-sizing or docking.
 =====================================================
#>

# 1. PASTE YOUR UI BACKGROUND CODE HERE
$Base64UI = "PASTE_UI_BACKGROUND_CODE_HERE"

# 2. PASTE YOUR WALLPAPER/LOGO CODE HERE
$Base64Wall = "PASTE_WALLPAPER_CODE_HERE"

# 3. SET YOUR CUSTOM SCRIPT LINK HERE
$TestLink = "https://raw.githubusercontent.com/username/repo/main/script.ps1" 

# ---------- ADMIN CHECK ----------
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Start-Process powershell "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    exit
}

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName Microsoft.VisualBasic

# ---------- THEME ----------
$Theme = @{
    PanelBg    = [System.Drawing.Color]::FromArgb(100, 0, 0, 0) 
    NeonBlue   = [System.Drawing.ColorTranslator]::FromHtml("#00F0FF") 
    NeonPurple = [System.Drawing.ColorTranslator]::FromHtml("#BD00FF") 
    TextWhite  = [System.Drawing.ColorTranslator]::FromHtml("#FFFFFF")
    ActiveTab  = [System.Drawing.Color]::FromArgb(80, 0, 240, 255) 
    InactiveTab= [System.Drawing.Color]::FromArgb(40, 0, 0, 0)   
}

# ---------- HELPER FUNCTIONS ----------
function Get-ImageFromBase64($string) {
    try {
        if ($string -match "PASTE_" -or [string]::IsNullOrWhiteSpace($string)) { return $null }
        $bytes = [Convert]::FromBase64String($string)
        $ms = New-Object System.IO.MemoryStream(,$bytes)
        return [System.Drawing.Image]::FromStream($ms)
    } catch { return $null }
}

function New-SciFiInput ($pnl, $x, $y, $w, $val) {
    $txt = New-Object System.Windows.Forms.TextBox
    $txt.Text = $val; $txt.Font = New-Object System.Drawing.Font("Consolas", 11)
    $txt.BackColor = [System.Drawing.Color]::Black; $txt.ForeColor = $Theme.NeonBlue; $txt.BorderStyle = 'None'
    $txt.Location = "$x, $y"; $txt.Width = $w
    $txt.Select($txt.Text.Length, 0)
    $pnl.Controls.Add($txt)
    $line = New-Object System.Windows.Forms.Panel; $line.BackColor = $Theme.NeonBlue; $line.Location = "$x, $($y + 18)"; $line.Size = "$w, 2"
    $pnl.Controls.Add($line)
    return $txt
}

function New-SciFiButton ($parent, $text, $x, $y, $w, $h, $action) {
    $btn = New-Object System.Windows.Forms.Button
    $btn.Text = $text; $btn.Font = New-Object System.Drawing.Font("Consolas", 9, [System.Drawing.FontStyle]::Bold)
    $btn.ForeColor = $Theme.TextWhite; $btn.BackColor = $Theme.InactiveTab
    $btn.FlatStyle = 'Flat'; $btn.FlatAppearance.BorderSize = 1; $btn.FlatAppearance.BorderColor = $Theme.NeonBlue
    $btn.Location = "$x, $y"; $btn.Size = "$w, $h"; $btn.Cursor = [System.Windows.Forms.Cursors]::Hand
    $btn.Add_Click($action)
    $parent.Controls.Add($btn)
    return $btn
}

function New-SciFiPanel ($parent, $title, $y, $height) {
    $pnl = New-Object System.Windows.Forms.Panel
    $pnl.Location = "150, $y"; $pnl.Size = "660, $height"; $pnl.BackColor = $Theme.PanelBg
    $stripe = New-Object System.Windows.Forms.Panel; $stripe.Size = "4, $height"; $stripe.Dock = 'Left'; $stripe.BackColor = $Theme.NeonPurple
    $pnl.Controls.Add($stripe)
    $lbl = New-Object System.Windows.Forms.Label; $lbl.Text = "[ $title ]"; $lbl.Font = New-Object System.Drawing.Font("Consolas", 10, [System.Drawing.FontStyle]::Bold)
    $lbl.ForeColor = $Theme.NeonBlue; $lbl.Location = "15, 10"; $lbl.AutoSize = $true
    $pnl.Controls.Add($lbl)
    $parent.Controls.Add($pnl)
    return $pnl
}

function New-Label ($pnl, $text, $x, $y) {
    $lbl = New-Object System.Windows.Forms.Label
    $lbl.Text = $text; $lbl.ForeColor = "White"; $lbl.Location = "$x, $y"; $lbl.AutoSize = $true; $lbl.Font="Consolas, 10"
    $pnl.Controls.Add($lbl)
}

function Connect-Map ($path, $user, $pass, $name) {
    $target = $path.Replace('\', '')
    cmdkey /add:$target /user:$user /pass:$pass | Out-Null
    Start-Process "explorer.exe" $path
    [System.Windows.Forms.MessageBox]::Show("$name CONNECTED & SAVED", "TGS SUCCESS")
}

function Remove-Map ($path, $name) {
    $target = $path.Replace('\', '')
    cmdkey /delete:$target | Out-Null
    net use $path /delete /y | Out-Null
    [System.Windows.Forms.MessageBox]::Show("$name DISCONNECTED", "TGS INFO")
}

function Switch-Tab ($activePanel, $activeBtn) {
    $pgSystem.Visible=$false; $pgNetwork.Visible=$false; $pgSoftware.Visible=$false; $pgSecurity.Visible=$false
    $btnTabSys.BackColor=$Theme.InactiveTab; $btnTabNet.BackColor=$Theme.InactiveTab; $btnTabSoft.BackColor=$Theme.InactiveTab; $btnTabSec.BackColor=$Theme.InactiveTab
    $activePanel.Visible = $true
    $activeBtn.BackColor = $Theme.ActiveTab
}

# ---------- FORM SETUP ----------
$form = New-Object System.Windows.Forms.Form
$form.Text = "TGS INIT"; $form.Size = '960, 650'; $form.StartPosition = 'CenterScreen'; $form.FormBorderStyle = 'None' 

$bgImage = Get-ImageFromBase64 $Base64UI
if ($bgImage) { $form.BackgroundImage = $bgImage; $form.BackgroundImageLayout = "Stretch" }
else { $form.BackColor = [System.Drawing.Color]::FromArgb(10,10,20) }

$header = New-Object System.Windows.Forms.Panel; $header.Dock = 'Top'; $header.Height = 50; $header.BackColor = [System.Drawing.Color]::FromArgb(180, 0, 0, 0)
$form.Controls.Add($header)
$headLine = New-Object System.Windows.Forms.Panel; $headLine.Dock='Bottom'; $headLine.Height=2; $headLine.BackColor=$Theme.NeonBlue
$header.Controls.Add($headLine)
$isDragging = $false; $dragPoint = $null
$header.Add_MouseDown({ $script:isDragging = $true; $script:dragPoint = $_.Location })
$header.Add_MouseUp({ $script:isDragging = $false })
$header.Add_MouseMove({ if ($script:isDragging) { $form.Location = [System.Drawing.Point]::new($form.Location.X + ($_.X - $script:dragPoint.X), $form.Location.Y + ($_.Y - $script:dragPoint.Y)) } })

$lblTitle = New-Object System.Windows.Forms.Label
$lblTitle.Text = "T G S   S Y S T E M   I N I T I A L I Z A T I O N" 
$lblTitle.Font = New-Object System.Drawing.Font("Consolas", 14, [System.Drawing.FontStyle]::Bold)
$lblTitle.ForeColor = $Theme.TextWhite; $lblTitle.Location = "20, 15"; $lblTitle.AutoSize = $true
$header.Controls.Add($lblTitle)

$btnClose = New-Object System.Windows.Forms.Button
$btnClose.Text = "X"; $btnClose.FlatStyle = 'Flat'; $btnClose.FlatAppearance.BorderSize = 0
$btnClose.ForeColor = 'Red'; $btnClose.Font = 'Consolas, 14'; $btnClose.Location = "910, 0"; $btnClose.Size = "50, 48"
$btnClose.Add_Click({ $form.Close() })
$header.Controls.Add($btnClose)

# =====================================================
# PANELS SETUP
# =====================================================
$pgSystem   = New-Object System.Windows.Forms.Panel; $pgSystem.Dock='Fill'; $pgSystem.BackColor='Transparent'; $form.Controls.Add($pgSystem)
$pgNetwork  = New-Object System.Windows.Forms.Panel; $pgNetwork.Dock='Fill'; $pgNetwork.BackColor='Transparent'; $pgNetwork.Visible=$false; $form.Controls.Add($pgNetwork)
$pgSoftware = New-Object System.Windows.Forms.Panel; $pgSoftware.Dock='Fill'; $pgSoftware.BackColor='Transparent'; $pgSoftware.Visible=$false; $form.Controls.Add($pgSoftware)
$pgSecurity = New-Object System.Windows.Forms.Panel; $pgSecurity.Dock='Fill'; $pgSecurity.BackColor='Transparent'; $pgSecurity.Visible=$false; $form.Controls.Add($pgSecurity)

# =====================================================
# 1. SYSTEM TAB
# =====================================================
$pnlSys1 = New-SciFiPanel $pgSystem "IDENTITY_MODULE" 50 85
New-Label $pnlSys1 "PC NAME :" 30 48; $txtPC = New-SciFiInput $pnlSys1 130 46 240 "TGS_"
New-SciFiButton $pnlSys1 "APPLY NAME" 420 38 150 30 { if ($txtPC.Text -eq $env:COMPUTERNAME) { [System.Windows.Forms.MessageBox]::Show("This PC is already named '$($txtPC.Text)'") } else { Rename-Computer -NewName $txtPC.Text -Force; [System.Windows.Forms.MessageBox]::Show("SUCCESS: REBOOT REQUIRED.") } }

$pnlSys2 = New-SciFiPanel $pgSystem "TEMPORAL_SYNC" 150 85
New-Label $pnlSys2 "ZONE : INDIA (12HR)" 30 48
New-SciFiButton $pnlSys2 "SYNC TIME" 420 38 150 30 { tzutil /s "India Standard Time"; Set-ItemProperty -Path "HKCU:\Control Panel\International" -Name "sShortTime" -Value "h:mm tt" -Force; w32tm /resync | Out-Null; [System.Windows.Forms.MessageBox]::Show("TIME SET TO INDIA (IST)") }

$pnlSys3 = New-SciFiPanel $pgSystem "VISUAL_CORE" 250 85
New-Label $pnlSys3 "WALLPAPER + LOCK : SET NEW" 30 48
New-SciFiButton $pnlSys3 "APPLY VISUALS" 420 38 150 30 {
    $wallImg = Get-ImageFromBase64 $Base64Wall
    if ($wallImg) { $imgPath = "C:\TGS_Assets\TGS_Wallpaper.jpg"; if (-not (Test-Path "C:\TGS_Assets")) { New-Item "C:\TGS_Assets" -Type Directory -Force | Out-Null }; $wallImg.Save($imgPath, [System.Drawing.Imaging.ImageFormat]::Jpeg); Set-ItemProperty -Path "HKCU:\Control Panel\Desktop" -Name Wallpaper -Value $imgPath -Force; RUNDLL32.EXE user32.dll, UpdatePerUserSystemParameters; [System.Windows.Forms.MessageBox]::Show("VISUALS APPLIED") }
}

$pnlRDP = New-SciFiPanel $pgSystem "REMOTE_ACCESS_CONTROL" 350 85
New-Label $pnlRDP "RDP STATUS : CHECKING" 30 48
New-SciFiButton $pnlRDP "ENABLE RDP" 300 38 120 30 { Set-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server" -Name "fDenyTSConnections" -Value 0; Enable-NetFirewallRule -DisplayGroup "Remote Desktop" -ErrorAction SilentlyContinue }
New-SciFiButton $pnlRDP "DISABLE RDP" 440 38 120 30 { Set-ItemProperty "HKLM:\SYSTEM\CurrentControlSet\Control\Terminal Server" -Name "fDenyTSConnections" -Value 1 }

# --- STORAGE PANEL ---
$pnlStorage = New-SciFiPanel $pgSystem "NETWORK_STORAGE" 450 120
New-Label $pnlStorage "TGS SERVER (121):" 40 40
$btnTGSConn = New-SciFiButton $pnlStorage "CONNECT" 200 35 100 30 { Connect-Map "\\174.156.5.121" "tgsuser121" "@121#Sgt" "TGS SERVER" }; $btnTGSConn.ForeColor="Lime"; $btnTGSConn.FlatAppearance.BorderColor="Lime"
$btnTGSRem  = New-SciFiButton $pnlStorage "REMOVE"  310 35 100 30 { Remove-Map "\\174.156.5.121" "TGS SERVER" }; $btnTGSRem.ForeColor="Red"; $btnTGSRem.FlatAppearance.BorderColor="Red"

New-Label $pnlStorage "NAS STORAGE (4.3):" 40 80
$btnNASConn = New-SciFiButton $pnlStorage "CONNECT" 200 75 100 30 { Connect-Map "\\174.156.4.3" "prog" "Triveni$456" "NAS STORAGE" }; $btnNASConn.ForeColor="Lime"; $btnNASConn.FlatAppearance.BorderColor="Lime"
$btnNASRem  = New-SciFiButton $pnlStorage "REMOVE"  310 75 100 30 { Remove-Map "\\174.156.4.3" "NAS STORAGE" }; $btnNASRem.ForeColor="Red"; $btnNASRem.FlatAppearance.BorderColor="Red"

# =====================================================
# 2. NETWORK TAB
# =====================================================
$pnlNet1 = New-SciFiPanel $pgNetwork "NETWORK_ADAPTER" 90 85
New-Label $pnlNet1 "ACTIVE ADAPTER : AUTO DETECT" 30 48; 

$pnlNet2 = New-SciFiPanel $pgNetwork "IPV4_CONFIGURATION" 200 120
New-Label $pnlNet2 "IP ADDRESS :" 30 40;  $txtIP  = New-SciFiInput $pnlNet2 150 38 130 "174.156.5."
New-Label $pnlNet2 "SUBNET :"     310 40;  $txtSub = New-SciFiInput $pnlNet2 400 38 130 "255.255.252.0"
New-Label $pnlNet2 "GATEWAY :"    30 80;   $txtGW  = New-SciFiInput $pnlNet2 150 78 130 "174.156.5.11"
New-Label $pnlNet2 "DNS :"        310 80;  $txtDNS = New-SciFiInput $pnlNet2 400 78 130 "8.8.8.8, 8.8.4.4"

$pnlNet3 = New-SciFiPanel $pgNetwork "NETWORK_ACTIONS" 340 85
New-Label $pnlNet3 "PING (ICMP) :" 30 48
New-SciFiButton $pnlNet3 "APPLY NETWORK" 400 38 170 30 { [System.Windows.Forms.MessageBox]::Show("Applying IP Settings...") }

# =====================================================
# 3. SOFTWARE TAB (BUTTON FIX)
# =====================================================
$pnlSoft = New-SciFiPanel $pgSoftware "PACKAGE_MANAGER" 50 480

# ---- COLUMN 1: BASIC APPS ----
$grpBasic = New-Object System.Windows.Forms.GroupBox; $grpBasic.Text=" 1. BASIC APPS "; $grpBasic.ForeColor=$Theme.NeonBlue; $grpBasic.Font="Consolas, 9, style=Bold"; $grpBasic.Location="20, 30"; $grpBasic.Size="200, 240"; $pnlSoft.Controls.Add($grpBasic)
$chkListBasic = New-Object System.Windows.Forms.CheckedListBox; $chkListBasic.Dock='Fill'; $chkListBasic.BackColor="Black"; $chkListBasic.ForeColor="White"; $chkListBasic.BorderStyle='None'; $chkListBasic.CheckOnClick=$true
$chkListBasic.Items.AddRange(@(
    "Google Chrome", "7-Zip", "WinRAR", "Notepad++", "VS Code", "VLC Media", "Lightshot", 
    "Adobe Reader", "Firefox", "MS Teams", "Zoom"
))
$grpBasic.Controls.Add($chkListBasic)
$defaults = @("Google Chrome", "7-Zip", "WinRAR", "Notepad++", "VS Code", "VLC Media", "Lightshot")
for($i=0; $i -lt $chkListBasic.Items.Count; $i++) { if ($defaults -contains $chkListBasic.Items[$i]) { $chkListBasic.SetItemChecked($i, $true) } }

# ---- COLUMN 2: DEVELOPER ----
$grpDev = New-Object System.Windows.Forms.GroupBox; $grpDev.Text=" 2. DEVELOPER "; $grpDev.ForeColor=$Theme.NeonBlue; $grpDev.Font="Consolas, 9, style=Bold"; $grpDev.Location="230, 30"; $grpDev.Size="200, 240"; $pnlSoft.Controls.Add($grpDev)
$chkListDev = New-Object System.Windows.Forms.CheckedListBox; $chkListDev.Dock='Fill'; $chkListDev.BackColor="Black"; $chkListDev.ForeColor="White"; $chkListDev.BorderStyle='None'; $chkListDev.CheckOnClick=$true
$chkListDev.Items.AddRange(@("Git", "JDK 17", "Python", "Postman", "Node.js"))
$grpDev.Controls.Add($chkListDev)
$defaultsDev = @("Git", "JDK 17")
for($i=0; $i -lt $chkListDev.Items.Count; $i++) { if ($defaultsDev -contains $chkListDev.Items[$i]) { $chkListDev.SetItemChecked($i, $true) } }

# ---- COLUMN 3: MANUAL & LOG ----
$grpMan = New-Object System.Windows.Forms.GroupBox; $grpMan.Text=" 3. MANUAL SETUP "; $grpMan.ForeColor="Orange"; $grpMan.Font="Consolas, 9, style=Bold"; $grpMan.Location="440, 30"; $grpMan.Size="200, 115"; $pnlSoft.Controls.Add($grpMan)
$chkListMan = New-Object System.Windows.Forms.CheckedListBox; $chkListMan.Dock='Fill'; $chkListMan.BackColor="Black"; $chkListMan.ForeColor="Orange"; $chkListMan.BorderStyle='None'; $chkListMan.CheckOnClick=$true
$chkListMan.Items.AddRange(@(
    "Visual Studio 2022", "SQL Server 2019", "SQL Server 2022", 
    "SSMS 19", "MySQL 8.0", "MongoDB 7.0", "RabbitMQ + Erlang"
))
$grpMan.Controls.Add($chkListMan)

$grpLog = New-Object System.Windows.Forms.GroupBox; $grpLog.Text=" SYSTEM LOG "; $grpLog.ForeColor="Lime"; $grpLog.Font="Consolas, 9, style=Bold"; $grpLog.Location="440, 155"; $grpLog.Size="200, 115"; $pnlSoft.Controls.Add($grpLog)
$txtLog = New-Object System.Windows.Forms.ListBox; $txtLog.Dock='Fill'; $txtLog.BackColor="Black"; $txtLog.ForeColor="Lime"; $txtLog.BorderStyle='None'; $txtLog.Font="Consolas, 8"; $grpLog.Controls.Add($txtLog)

# ---- COMMAND DECK (WIDER BUTTONS) ----
$progressBar = New-Object System.Windows.Forms.ProgressBar; $progressBar.Location = "20, 280"; $progressBar.Size = "620, 5"; $progressBar.Style = 'Continuous'; $pnlSoft.Controls.Add($progressBar)

# ROW 1 (UTILITIES) - FIXED SPACING & WIDTH
New-SciFiButton $pnlSoft "ALL" 20 300 60 35 { for($i=0; $i -lt $chkListBasic.Items.Count; $i++) { $chkListBasic.SetItemChecked($i, $true) }; for($i=0; $i -lt $chkListDev.Items.Count; $i++) { $chkListDev.SetItemChecked($i, $true) } }
New-SciFiButton $pnlSoft "NONE" 90 300 60 35 { for($i=0; $i -lt $chkListBasic.Items.Count; $i++) { $chkListBasic.SetItemChecked($i, $false) }; for($i=0; $i -lt $chkListDev.Items.Count; $i++) { $chkListDev.SetItemChecked($i, $false) } }
New-SciFiButton $pnlSoft "DEFAULT" 160 300 80 35 { for($i=0; $i -lt $chkListBasic.Items.Count; $i++) { $chkListBasic.SetItemChecked($i, $defaults -contains $chkListBasic.Items[$i]) }; for($i=0; $i -lt $chkListDev.Items.Count; $i++) { $chkListDev.SetItemChecked($i, $defaultsDev -contains $chkListDev.Items[$i]) } }
New-SciFiButton $pnlSoft "CLR LOG" 250 300 80 35 { $txtLog.Items.Clear() }
$btnTest = New-SciFiButton $pnlSoft "TEST" 340 300 100 35 { if ($TestLink) { Start-Process powershell -ArgumentList "-NoProfile -Command `"irm $TestLink | iex`""; $txtLog.Items.Add("[ TEST ] Running Script...") } }
$btnTest.BackColor="Black"; $btnTest.ForeColor="White"; $btnTest.FlatAppearance.BorderColor="White"

# ROW 2 (ACTIONS) - WIDER
$btnOffice = New-SciFiButton $pnlSoft "INSTALL OFFICE 2019" 20 340 200 40 { 
    $txtLog.Items.Add("[ MS ] Downloading Office..."); $progressBar.Style='Marquee'; [System.Windows.Forms.Application]::DoEvents()
    choco install office2019proplus -y; $progressBar.Style='Continuous'; $progressBar.Value=100; $txtLog.Items.Add("[ MS ] Office Installed.")
}
$btnOffice.ForeColor="Orange"; $btnOffice.FlatAppearance.BorderColor="Orange"

$script:StopInstall = $false
New-SciFiButton $pnlSoft "INSTALL START" 230 340 300 40 {
    $script:StopInstall = $false; $progressBar.Value = 0; $progressBar.Style = 'Continuous'; $txtLog.Items.Clear(); $txtLog.Items.Add("[ INIT ] Scanning...")
    $apps = @()
    foreach ($item in $chkListBasic.CheckedItems) { switch ($item) { "Google Chrome"{$apps+="googlechrome"} "7-Zip"{$apps+="7zip"} "WinRAR"{$apps+="winrar"} "Notepad++"{$apps+="notepadplusplus"} "VS Code"{$apps+="vscode"} "Lightshot"{$apps+="lightshot"} "VLC Media"{$apps+="vlc"} "Adobe Reader"{$apps+="adobereader"} "Firefox"{$apps+="firefox"} "Zoom"{$apps+="zoom"} "MS Teams"{$apps+="microsoft-teams"} } }
    foreach ($item in $chkListDev.CheckedItems) { switch ($item) { "Git"{$apps+="git"} "JDK 17"{$apps+="openjdk17"} "Python"{$apps+="python"} "Postman"{$apps+="postman"} "Node.js"{$apps+="nodejs"} } }
    foreach ($item in $chkListMan.CheckedItems) {
        $txtLog.Items.Add(">> MANUAL: $item")
        switch ($item) {
            "Visual Studio 2022" { $txtLog.Items.Add("[ NOTE ] Select: ASP.NET, Azure, Node, Desktop"); choco install visualstudio2022community --package-parameters "--add Microsoft.VisualStudio.Workload.NetWeb --add Microsoft.VisualStudio.Workload.Azure --add Microsoft.VisualStudio.Workload.Node --add Microsoft.VisualStudio.Workload.ManagedDesktop --add Microsoft.VisualStudio.Workload.NetCrossPlat --includeRecommended --passive --norestart" -y }
            "SQL Server 2019" { $txtLog.Items.Add("[ NOTE ] Mixed Mode | Pass: triveni@123"); choco install sql-server-2019 -y }
            "SQL Server 2022" { $txtLog.Items.Add("[ NOTE ] Mixed Mode | Pass: triveni@123"); choco install sql-server-2022 -y }
            "SSMS 19" { $txtLog.Items.Add("[ NOTE ] Run as Admin"); choco install sql-server-management-studio -y }
            "MySQL 8.0" { $txtLog.Items.Add("[ NOTE ] Install 8.0.35"); choco install mysql -y }
            "MongoDB 7.0" { $txtLog.Items.Add("[ NOTE ] Install Compass"); choco install mongodb -y }
            "RabbitMQ + Erlang" { $txtLog.Items.Add("[ NOTE ] Erlang 25.1 -> RabbitMQ 3.11"); choco install erlang -y; choco install rabbitmq -y }
        }
    }
    if ($apps.Count -gt 0) {
        if (-not (Get-Command choco -ErrorAction SilentlyContinue)) { $txtLog.Items.Add("[ INFO ] Installing Choco..."); [System.Windows.Forms.Application]::DoEvents(); Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1')) }
        $step = 100 / ($apps.Count + 1); $current = 0
        foreach ($pkg in $apps) { if ($script:StopInstall) { $txtLog.Items.Add("[ STOP ] User Halted"); return }; $check = choco list --local-only --exact $pkg; if ($check -match "$pkg") { $txtLog.Items.Add("[ OK ] $pkg Checked") } else { $txtLog.Items.Add("[ .... ] Installing $pkg"); [System.Windows.Forms.Application]::DoEvents(); choco install $pkg -y --no-progress | Out-Null; $txtLog.Items.Add("[ DONE ] $pkg Installed") }; $txtLog.TopIndex = $txtLog.Items.Count - 1; $current += $step; if ($current -gt 100) { $current = 100 }; $progressBar.Value = [int]$current }
    }
    $progressBar.Value = 100; $txtLog.Items.Add("[ FINISH ] Tasks Done.")
}

New-SciFiButton $pnlSoft "HALT" 540 340 100 40 { $script:StopInstall = $true; $txtLog.Items.Add("[ STOP ] Halting...") } | ForEach-Object { $_.ForeColor="Red"; $_.FlatAppearance.BorderColor="Red" }

# =====================================================
# 4. SECURITY TAB
# =====================================================
$pnlSec2 = New-SciFiPanel $pgSecurity "DATA_LEAK_PREVENTION" 90 120
New-Label $pnlSec2 "RDP CLIPBOARD :" 30 40
New-SciFiButton $pnlSec2 "BLOCK" 200 30 100 30 { New-Item "HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services" -Force | Out-Null; Set-ItemProperty "HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services" -Name "DisableClipboardRedirection" -Value 1; [System.Windows.Forms.MessageBox]::Show("RDP CLIPBOARD BLOCKED") }
New-SciFiButton $pnlSec2 "ALLOW" 320 30 100 30 { Set-ItemProperty "HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services" -Name "DisableClipboardRedirection" -Value 0; [System.Windows.Forms.MessageBox]::Show("RDP CLIPBOARD ALLOWED") }

New-Label $pnlSec2 "BLUETOOTH FILE:" 30 80
New-SciFiButton $pnlSec2 "BLOCK" 200 70 100 30 { New-Item "HKLM:\SOFTWARE\Policies\Microsoft\Windows\Bluetooth" -Force | Out-Null; Set-ItemProperty "HKLM:\SOFTWARE\Policies\Microsoft\Windows\Bluetooth" -Name "DisableFileTransfer" -Value 1; [System.Windows.Forms.MessageBox]::Show("BT TRANSFER BLOCKED") }
New-SciFiButton $pnlSec2 "ALLOW" 320 70 100 30 { Set-ItemProperty "HKLM:\SOFTWARE\Policies\Microsoft\Windows\Bluetooth" -Name "DisableFileTransfer" -Value 0; [System.Windows.Forms.MessageBox]::Show("BT TRANSFER ALLOWED") }

# =====================================================
# NAVIGATION BAR
# =====================================================
$navBar = New-Object System.Windows.Forms.Panel; $navBar.Dock = 'Bottom'; $navBar.Height = 60; $navBar.BackColor = [System.Drawing.Color]::FromArgb(180, 0, 0, 0)
$form.Controls.Add($navBar)

# Reordered Buttons: Sys -> Net -> Soft -> Sec
$btnTabSys  = New-SciFiButton $navBar "SYSTEM"  20 10 160 40 { Switch-Tab $pgSystem $btnTabSys }
$btnTabNet  = New-SciFiButton $navBar "NETWORK" 190 10 160 40 { Switch-Tab $pgNetwork $btnTabNet }
$btnTabSoft = New-SciFiButton $navBar "SOFTWARE" 360 10 160 40 { Switch-Tab $pgSoftware $btnTabSoft }
$btnTabSec  = New-SciFiButton $navBar "SECURITY" 530 10 160 40 { Switch-Tab $pgSecurity $btnTabSec }

$btnTabSys.BackColor = $Theme.ActiveTab # Default

$btnReboot = New-SciFiButton $navBar "REBOOT" 780 10 140 40 { Restart-Computer -Force }
$btnReboot.ForeColor = 'Red'; $btnReboot.FlatAppearance.BorderColor = 'Red'

$form.ShowDialog() | Out-Null