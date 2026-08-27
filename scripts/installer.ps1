# =============================================================================
# TGS SYSTEM CORE // MASTER INTERFACE V12 (SQL 2019 CUSTOM CONFIG)
# LOCATION: DESKTOP EXECUTION ONLY
# PASSWORD PROTOCOL: triveni@123
# =============================================================================

# --- 1. ELEVATION CHECK ---
if (!([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Start-Process powershell.exe "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    Exit
}

# --- 2. INITIALIZATION ---
$ErrorActionPreference = "SilentlyContinue"
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$GlobalPassword = "triveni@123"

# Log Setup
$DesktopPath = [Environment]::GetFolderPath("Desktop")
$LogFolder = "$DesktopPath\TGS_Temp_Logs"
if (!(Test-Path $LogFolder)) { New-Item -Path $LogFolder -ItemType Directory | Out-Null }
$LogFile = "$LogFolder\Install_Log_$(Get-Date -Format 'yyyy-MM-dd').txt"

# =============================================================================
# SOFTWARE MANIFEST
# =============================================================================
$SoftwareList = @(
    @{ Rank = "01"; Name = "VISUAL STUDIO ENT 2022"; Type = "EXE"; Path = "\\174.156.4.3\fjt\Guru\Visual Studio All\VS 2022\Visual Studio Enterprise.exe"; CheckPath = "C:\Program Files\Microsoft Visual Studio\2022\Enterprise\Common7\IDE\devenv.exe"; Reboot = "YES"; Args = "--passive --norestart --wait --productKey VHF9H-NXBBB-638P6-6JHCY-88JWH --add Microsoft.VisualStudio.Workload.NetWeb --add Microsoft.VisualStudio.Workload.Azure --add Microsoft.VisualStudio.Workload.Node --add Microsoft.VisualStudio.Workload.NetCrossPlat --add Microsoft.VisualStudio.Workload.ManagedDesktop --includeRecommended" },
    
    # UPDATED: Changed Type to 'Custom_SQL2019' to trigger the new popup
    @{ Rank = "02"; Name = "SQL SERVER 2019 (CONFIG)"; Type = "Custom_SQL2019"; Path = "\\174.156.4.3\fjt\Guru\SQL Server 2019 15.0\SQL2019\ExpressAdv_ENU\SETUP.EXE"; CheckPath = "C:\Program Files\Microsoft SQL Server\MSSQL15.SQLEXPRESS"; Reboot = "NO"; Args = "" },
    
    @{ Rank = "03"; Name = "SQL SERVER 2022 EXPRESS"; Type = "EXE"; Path = "\\174.156.4.3\fjt\Guru\SQL Server 2022 16.0\SQL2022-SSEI-Expr.exe"; CheckPath = "C:\Program Files\Microsoft SQL Server\MSSQL16.MSSQLSERVER"; Reboot = "LIKELY"; Args = "/Action=Install /IACCEPTSQLSERVERLICENSETERMS /Quiet" },
    @{ Rank = "04"; Name = "SSMS 20.0 (MGMT STUDIO)"; Type = "EXE"; Path = "\\174.156.4.3\fjt\Guru\SSMS\SSMS-Setup-ENU 2024 20.0.exe"; CheckPath = "C:\Program Files (x86)\Microsoft SQL Server Management Studio 20"; Reboot = "NO"; Args = "/Quiet /NoRestart" },
    @{ Rank = "05"; Name = "MYSQL SERVER 8.0.35"; Type = "Custom_MySQL"; Path = "\\174.156.4.3\fjt\Guru\MySQL installer 8.0.35\mysql-installer-community-8.0.35.0 (Compulsory).msi"; CheckPath = "C:\Program Files\MySQL\MySQL Server 8.0"; Reboot = "NO"; Args = "" },
    @{ Rank = "06"; Name = "MONGODB 7.0.4"; Type = "MSI"; Path = "\\174.156.4.3\fjt\Guru\MongoDB 7.0.4\mongodb-windows-x86_64-7.0.4-signed.msi"; CheckPath = "C:\Program Files\MongoDB\Server\7.0"; Reboot = "NO"; Args = "/quiet /norestart ADDLOCAL=`"ServerService,Client`"" },
    @{ Rank = "07"; Name = "RABBITMQ + ERLANG OTP"; Type = "PS1"; Path = "\\174.156.4.3\fjt\Guru\RabbitMQ\RabbitMQ-FullOfflineInstall.ps1"; CheckPath = "C:\Program Files\RabbitMQ Server"; Reboot = "NO"; Args = "" },
    @{ Rank = "08"; Name = "TIGHT VNC VIEWER"; Type = "MSI"; Path = "\\174.156.4.3\fjt\Guru\TightVNC\tightvnc-2.8.81-gpl-setup-64bit.msi"; CheckPath = "C:\Program Files\TightVNC"; Reboot = "NO"; Args = "/quiet /norestart" }
)

# =============================================================================
# GUI BUILDER
# =============================================================================
$C_Back   = [System.Drawing.ColorTranslator]::FromHtml("#050505")
$C_Text   = [System.Drawing.ColorTranslator]::FromHtml("#00FFFF")
$C_Green  = [System.Drawing.ColorTranslator]::FromHtml("#00FF00")
$C_Red    = [System.Drawing.ColorTranslator]::FromHtml("#FF3333")
$C_Dark   = [System.Drawing.ColorTranslator]::FromHtml("#111111")
$C_Select = [System.Drawing.ColorTranslator]::FromHtml("#003333")

$Form = New-Object System.Windows.Forms.Form
$Form.Text = "TGS SYSTEM CORE // MASTER CONTROL V12"
$Form.Size = New-Object System.Drawing.Size(1200, 800)
$Form.StartPosition = "CenterScreen"
$Form.BackColor = $C_Back
$Form.ForeColor = $C_Text

$Header = New-Object System.Windows.Forms.Label
$Header.Dock = "Top"
$Header.Height = 35
$Header.Text = "  SYSTEM_CORE :: INSTALLER_MODE [ADMIN]"
$Header.TextAlign = "MiddleLeft"
$Header.BackColor = $C_Dark
$Header.ForeColor = $C_Green
$Header.Font = New-Object System.Drawing.Font("Consolas", 12, [System.Drawing.FontStyle]::Bold)
$Form.Controls.Add($Header)

$Split = New-Object System.Windows.Forms.SplitContainer
$Split.Dock = "Fill"
$Split.Orientation = "Horizontal"
$Split.SplitterDistance = 450
$Split.BackColor = $C_Text
$Form.Controls.Add($Split)

$Grid = New-Object System.Windows.Forms.DataGridView
$Grid.Dock = "Fill"
$Grid.BackgroundColor = $C_Back
$Grid.BorderStyle = "None"
$Grid.CellBorderStyle = "SingleHorizontal"
$Grid.GridColor = [System.Drawing.ColorTranslator]::FromHtml("#222222")
$Grid.RowHeadersVisible = $false
$Grid.AllowUserToAddRows = $false
$Grid.RowTemplate.Height = 45
$Grid.EnableHeadersVisualStyles = $false

$Grid.ColumnHeadersDefaultCellStyle.BackColor = $C_Dark
$Grid.ColumnHeadersDefaultCellStyle.ForeColor = $C_Text
$Grid.ColumnHeadersDefaultCellStyle.Font = New-Object System.Drawing.Font("Consolas", 10, [System.Drawing.FontStyle]::Bold)
$Grid.ColumnHeadersHeight = 40
$Grid.DefaultCellStyle.BackColor = $C_Back
$Grid.DefaultCellStyle.ForeColor = $C_Green
$Grid.DefaultCellStyle.Font = New-Object System.Drawing.Font("Consolas", 10)
$Grid.DefaultCellStyle.SelectionBackColor = $C_Select
$Grid.DefaultCellStyle.SelectionForeColor = $C_Text

$Grid.Columns.Add("Rank", "ID") | Out-Null
$Grid.Columns[0].Width = 40
$Grid.Columns[0].DefaultCellStyle.Alignment = "MiddleCenter"

$Grid.Columns.Add("Name", "MODULE NAME") | Out-Null
$Grid.Columns[1].AutoSizeMode = "Fill"

$Grid.Columns.Add("Status", "STATUS") | Out-Null
$Grid.Columns[2].Width = 100
$Grid.Columns[2].DefaultCellStyle.Alignment = "MiddleCenter"

$btnInst = New-Object System.Windows.Forms.DataGridViewButtonColumn
$btnInst.HeaderText = "ACTION"
$btnInst.Text = "[ DEPLOY ]"
$btnInst.UseColumnTextForButtonValue = $true
$btnInst.FlatStyle = "Flat"
$btnInst.DefaultCellStyle.BackColor = $C_Back
$btnInst.DefaultCellStyle.ForeColor = $C_Green
$Grid.Columns.Add($btnInst) | Out-Null
$Grid.Columns[3].Width = 130

$btnNuke = New-Object System.Windows.Forms.DataGridViewButtonColumn
$btnNuke.HeaderText = "HAZARD"
$btnNuke.Text = "[ REMOVE ]"
$btnNuke.UseColumnTextForButtonValue = $true
$btnNuke.FlatStyle = "Flat"
$btnNuke.DefaultCellStyle.BackColor = $C_Back
$btnNuke.DefaultCellStyle.ForeColor = $C_Red
$Grid.Columns.Add($btnNuke) | Out-Null
$Grid.Columns[4].Width = 130

$Split.Panel1.Controls.Add($Grid)

$LogPanel = New-Object System.Windows.Forms.Panel
$LogPanel.Dock = "Fill"
$Split.Panel2.Controls.Add($LogPanel)

$ProgressBar = New-Object System.Windows.Forms.ProgressBar
$ProgressBar.Dock = "Top"
$ProgressBar.Height = 10
$ProgressBar.Style = "Blocks"
$LogPanel.Controls.Add($ProgressBar)

$Log = New-Object System.Windows.Forms.RichTextBox
$Log.Dock = "Fill"
$Log.BackColor = $C_Back
$Log.ForeColor = $C_Green
$Log.Font = New-Object System.Drawing.Font("Lucida Console", 9)
$Log.BorderStyle = "None"
$Log.ReadOnly = $true
$LogPanel.Controls.Add($Log)
$Log.BringToFront()

# =============================================================================
# LOGIC CORE
# =============================================================================
function Write-Log($msg, $color="Cyan") {
    $timestamp = Get-Date -Format "HH:mm:ss"
    $Log.SelectionStart = $Log.TextLength
    $Log.SelectionLength = 0
    switch ($color) {
        "Cyan"   { $Log.SelectionColor = $C_Text }
        "Green"  { $Log.SelectionColor = $C_Green }
        "Red"    { $Log.SelectionColor = $C_Red }
        "Yellow" { $Log.SelectionColor = [System.Drawing.Color]::Yellow }
    }
    $Log.AppendText("[$timestamp] $msg`r`n")
    $Log.ScrollToCaret()
    $FileMsg = "[$timestamp] $msg"
    Add-Content -Path $LogFile -Value $FileMsg
    [System.Windows.Forms.Application]::DoEvents()
}

function Refresh-Grid {
    $Grid.Rows.Clear()
    foreach ($item in $SoftwareList) {
        $row = $Grid.Rows.Add()
        $Grid.Rows[$row].Cells[0].Value = $item.Rank
        $Grid.Rows[$row].Cells[1].Value = $item.Name
        if (Test-Path $item.CheckPath) {
            $Grid.Rows[$row].Cells[2].Value = "ONLINE"
            $Grid.Rows[$row].Cells[2].Style.ForeColor = $C_Green
        } else {
            $Grid.Rows[$row].Cells[2].Value = "OFFLINE"
            $Grid.Rows[$row].Cells[2].Style.ForeColor = [System.Drawing.Color]::Gray
        }
        $Grid.Rows[$row].Tag = $item
    }
    $Grid.ClearSelection()
}

# --- NEW SQL 2019 CUSTOM POPUP FUNCTION ---
function Run-SQL2019-Custom($path) {
    # 1. Create Popup Form
    $sd = New-Object System.Windows.Forms.Form
    $sd.Text = "INSTANCE CONFIGURATION"
    $sd.Size = New-Object System.Drawing.Size(450, 300)
    $sd.StartPosition = "CenterScreen"
    $sd.BackColor = $C_Back
    $sd.ForeColor = $C_Text
    $sd.FormBorderStyle = "FixedDialog"
    $sd.MaximizeBox = $false

    $lbl = New-Object System.Windows.Forms.Label
    $lbl.Text = "SELECT INSTANCE TYPE:"
    $lbl.Location = New-Object System.Drawing.Point(30, 20)
    $lbl.AutoSize = $true
    $lbl.Font = New-Object System.Drawing.Font("Consolas", 11, [System.Drawing.FontStyle]::Bold)
    $sd.Controls.Add($lbl)

    $rbDefault = New-Object System.Windows.Forms.RadioButton
    $rbDefault.Text = "DEFAULT INSTANCE (MSSQLSERVER)"
    $rbDefault.Location = New-Object System.Drawing.Point(35, 60)
    $rbDefault.Width = 350
    $rbDefault.Checked = $true
    $rbDefault.Font = New-Object System.Drawing.Font("Consolas", 10)
    $sd.Controls.Add($rbDefault)

    $rbNamed = New-Object System.Windows.Forms.RadioButton
    $rbNamed.Text = "NAMED INSTANCE (Custom)"
    $rbNamed.Location = New-Object System.Drawing.Point(35, 90)
    $rbNamed.Width = 350
    $rbNamed.Font = New-Object System.Drawing.Font("Consolas", 10)
    $sd.Controls.Add($rbNamed)

    $lblSub = New-Object System.Windows.Forms.Label
    $lblSub.Text = "Instance Name (If Named):"
    $lblSub.Location = New-Object System.Drawing.Point(55, 125)
    $lblSub.AutoSize = $true
    $lblSub.ForeColor = "Gray"
    $sd.Controls.Add($lblSub)

    $txtNamed = New-Object System.Windows.Forms.TextBox
    $txtNamed.Text = "SQLEXPRESS"
    $txtNamed.Location = New-Object System.Drawing.Point(55, 150)
    $txtNamed.Width = 200
    $txtNamed.BackColor = $C_Dark
    $txtNamed.ForeColor = $C_Text
    $sd.Controls.Add($txtNamed)

    $btnOk = New-Object System.Windows.Forms.Button
    $btnOk.Text = "DEPLOY CONFIG"
    $btnOk.DialogResult = "OK"
    $btnOk.Location = New-Object System.Drawing.Point(120, 200)
    $btnOk.Width = 150
    $btnOk.Height = 40
    $btnOk.FlatStyle = "Flat"
    $btnOk.BackColor = $C_Select
    $btnOk.ForeColor = $C_Green
    $sd.Controls.Add($btnOk)

    # 2. Show Popup and Get Result
    if ($sd.ShowDialog() -eq "OK") {
        $ProgressBar.Style = "Marquee"
        
        # Determine Instance Argument
        if ($rbDefault.Checked) {
            $InstanceArg = "/INSTANCENAME=MSSQLSERVER"
            Write-Log "Configuring as DEFAULT INSTANCE..." "Yellow"
        } else {
            $Name = $txtNamed.Text
            $InstanceArg = "/INSTANCENAME=$Name"
            Write-Log "Configuring as NAMED INSTANCE: $Name" "Yellow"
        }

        # 3. Build Arguments based on your Screenshots
        # - /FEATURES=SQLEngine,Replication (From Image 4)
        # - /BROWSERSVCSTARTUPTYPE="Automatic" (From Image 6)
        # - /SQLSVCINSTANTFILEINIT="True" (Volume Maintenance - Image 6)
        # - /SECURITYMODE=SQL /SAPWD... (Mixed Mode - Image 7)
        # - /SQLSYSADMINACCOUNTS="BUILTIN\ADMINISTRATORS" (Add Current User - Image 7)
        
        $FinalArgs = "/Q /ACTION=Install /IACCEPTSQLSERVERLICENSETERMS /FEATURES=SQLEngine,Replication $InstanceArg /SQLSVCINSTANTFILEINIT=`"True`" /BROWSERSVCSTARTUPTYPE=`"Automatic`" /SECURITYMODE=SQL /SAPWD=`"$GlobalPassword`" /SQLSYSADMINACCOUNTS=`"BUILTIN\ADMINISTRATORS`""
        
        Write-Log "Executing Installer with Custom Config..." "Cyan"
        Start-Process $path -ArgumentList $FinalArgs -Wait
        Write-Log "SQL Server 2019 Setup Finished." "Green"
        $ProgressBar.Style = "Blocks"
        $ProgressBar.Value = 100
    }
}

function Run-MySQL($path) {
    $ProgressBar.Style = "Marquee"
    # Pre-Check Kill
    Stop-Service "MySQL80" -Force -ErrorAction SilentlyContinue
    sc.exe delete "MySQL80" | Out-Null
    
    Write-Log "Stage 1: Installing Base MSI..." "Yellow"
    Start-Process "msiexec.exe" "/i `"$path`" /quiet /norestart" -Wait
    
    $tool = "${env:ProgramFiles(x86)}\MySQL\MySQL Installer for Windows\MySQLInstallerConsole.exe"
    if (!(Test-Path $tool)) { $tool = "${env:ProgramFiles}\MySQL\MySQL Installer for Windows\MySQLInstallerConsole.exe" }
    
    if (Test-Path $tool) {
        Write-Log "Stage 2: Installing Server Binaries..." "Yellow"
        Start-Process $tool "install server;8.0.35;x64 workbench;8.0.34;x64 -silent" -Wait
        
        Write-Log "Pausing 20s for registration..." "Cyan"
        Start-Sleep -Seconds 20
        
        Write-Log "Stage 3: Configuring Service ($GlobalPassword)..." "Yellow"
        Start-Process $tool "config server --servicename=MySQL80 --accounts=`"root:$GlobalPassword`" --legacy-auth=true --startuptype=auto -silent" -Wait
        
        Write-Log "Verifying..." "Cyan"
        Start-Sleep -Seconds 2
        if (Get-Service "MySQL80" -ErrorAction SilentlyContinue) {
             Write-Log "SUCCESS: MySQL Online." "Green"
        } else {
             Write-Log "RETRYING CONFIG..." "Red"
             Start-Process $tool "config server --servicename=MySQL80 --root-password=$GlobalPassword --startuptype=auto -silent" -Wait
             if (Get-Service "MySQL80" -ErrorAction SilentlyContinue) { 
                Write-Log "SUCCESS: Recovered & Online." "Green" 
             } else {
                Write-Log "FAILURE: Service not created." "Red"
             }
        }
    } else { Write-Log "Error: MySQL Console Not Detected." "Red" }
    $ProgressBar.Style = "Blocks"
    $ProgressBar.Value = 100
}

function Deep-Clean($item) {
    $TargetName = $item.Name.Split(" ")[0]
    Write-Log ">>> INITIATING NUCLEAR CLEAN: $TargetName" "Red"
    $ProgressBar.Style = "Marquee"

    # 1. KILL
    Get-Process | Where {$_.Name -like "*$TargetName*"} | Stop-Process -Force -ErrorAction SilentlyContinue

    # 2. DELETE SERVICE
    if ($TargetName -eq "MYSQL") {
        Write-Log "Deleting Service MySQL80..." "Red"
        Stop-Service "MySQL80" -Force -ErrorAction SilentlyContinue
        sc.exe delete "MySQL80"
    }

    # 3. DELETE FILES
    $Paths = @("C:\Program Files", "C:\Program Files (x86)", "C:\ProgramData", "$env:LOCALAPPDATA", "$env:APPDATA", "C:\ProgramData\Microsoft\Windows\Start Menu\Programs")
    foreach ($P in $Paths) {
        if (Test-Path $P) {
            Get-ChildItem -Path $P -Filter "*$TargetName*" -Directory -ErrorAction SilentlyContinue | ForEach-Object {
                Write-Log "Erasing: $($_.FullName)" "Red"
                Remove-Item -Path $_.FullName -Recurse -Force -ErrorAction SilentlyContinue
            }
        }
    }

    # 4. REGISTRY
    $RegPaths = @("HKLM:\SOFTWARE", "HKLM:\SOFTWARE\WOW6432Node", "HKCU:\Software")
    foreach ($R in $RegPaths) {
         Get-ChildItem -Path $R -ErrorAction SilentlyContinue | Where-Object { $_.Name -like "*$TargetName*" } | ForEach-Object {
             Write-Log "Deleting Key: $($_.Name)" "Red"
             Remove-Item -Path $_.PSPath -Recurse -Force -ErrorAction SilentlyContinue
         }
    }
    
    if ($TargetName -eq "MYSQL") {
        $InstallerData = "C:\ProgramData\MySQL\MySQL Installer for Windows"
        if (Test-Path $InstallerData) { Remove-Item $InstallerData -Recurse -Force -ErrorAction SilentlyContinue }
    }

    Write-Log "CLEAN COMPLETE." "Red"
    $ProgressBar.Style = "Blocks"
    $ProgressBar.Value = 0
}

$Grid.Add_CellContentClick({
    param($s, $e)
    if ($e.RowIndex -lt 0) { return }
    $item = $Grid.Rows[$e.RowIndex].Tag
    
    # INSTALL BUTTON
    if ($e.ColumnIndex -eq 3) {
        if (!(Test-Path $item.Path)) { [System.Windows.Forms.MessageBox]::Show("SOURCE NOT FOUND:`n$($item.Path)"); return }
        $Grid.Enabled = $false
        
        Write-Log ">>> EXECUTING PROTOCOL: $($item.Name)" "Cyan"
        try {
            # === SPECIAL HANDLER FOR SQL 2019 ===
            if ($item.Type -eq "Custom_SQL2019") { 
                Run-SQL2019-Custom $item.Path 
            }
            elseif ($item.Type -eq "Custom_MySQL") { 
                Run-MySQL $item.Path 
            }
            elseif ($item.Type -eq "PS1") { 
                $ProgressBar.Style = "Marquee"
                Start-Process "powershell.exe" "-ExecutionPolicy Bypass -File `"$($item.Path)`"" -Wait 
                $ProgressBar.Style = "Blocks"; $ProgressBar.Value = 100
            }
            else { 
                $ProgressBar.Style = "Marquee"
                if ($item.Type -eq "MSI") { Start-Process "msiexec.exe" "/i `"$($item.Path)`" $($item.Args)" -Wait }
                else { Start-Process $item.Path $item.Args -Wait }
                $ProgressBar.Style = "Blocks"; $ProgressBar.Value = 100
            }
            Write-Log "PROCESS COMPLETE." "Green"
            if ($item.Reboot -eq "YES") { Write-Log "SYSTEM RESTART REQUIRED." "Yellow" }
        } catch { Write-Log "CRITICAL ERROR: $_" "Red" }
        finally { $Grid.Enabled = $true; Refresh-Grid }
    }
    
    # REMOVE BUTTON
    if ($e.ColumnIndex -eq 4) { 
        $Msg = "CONFIRM NUCLEAR CLEAN FOR: $($item.Name)`n`nThis will erase:`n- Windows Services`n- Program Files`n- Registry Keys`n`nARE YOU SURE?"
        if ([System.Windows.Forms.MessageBox]::Show($Msg, "HAZARD WARNING", "YesNo", "Warning") -eq "Yes") {
            Deep-Clean $item
            Refresh-Grid
        }
    }
})

# --- BOOT ---
Write-Log "SYSTEM CORE INITIALIZED..." "Cyan"
Write-Log "LOGGING TO: $LogFile" "Cyan"
Refresh-Grid
$Form.ShowDialog()