<#
.SYNOPSIS
    TGS Headless Installer Tool
#>
param (
    [switch]$List,
    [switch]$Install,
    [string]$AppID,
    [string]$Config
)

$ErrorActionPreference = "SilentlyContinue"
$NetworkPath = "\\174.156.4.3\fjt\Guru"

$SoftwareCatalog = @(
    [PSCustomObject]@{ ID = "chrome"; Name = "Google Chrome"; Type = "Choco"; Package = "googlechrome" },
    [PSCustomObject]@{ ID = "7zip"; Name = "7-Zip"; Type = "Choco"; Package = "7zip" },
    [PSCustomObject]@{ ID = "vscode"; Name = "Visual Studio Code"; Type = "Choco"; Package = "vscode" },
    [PSCustomObject]@{ ID = "npp"; Name = "Notepad++"; Type = "Choco"; Package = "notepadplusplus" },
    [PSCustomObject]@{ ID = "git"; Name = "Git"; Type = "Choco"; Package = "git" },
    [PSCustomObject]@{ ID = "nodejs"; Name = "Node.js"; Type = "Choco"; Package = "nodejs" },
    [PSCustomObject]@{ ID = "python"; Name = "Python 3"; Type = "Choco"; Package = "python" },
    [PSCustomObject]@{ ID = "vs2022"; Name = "Visual Studio Enterprise 2022"; Type = "Exe"; Path = "$NetworkPath\Visual Studio All\VS 2022\Visual Studio Enterprise.exe"; Args = "--passive --norestart --wait --productKey VHF9H-NXBBB-638P6-6JHCY-88JWH --add Microsoft.VisualStudio.Workload.NetWeb --add Microsoft.VisualStudio.Workload.Azure --add Microsoft.VisualStudio.Workload.Node --includeRecommended" },
    [PSCustomObject]@{ ID = "sql2019"; Name = "SQL Server 2019"; Type = "Custom_SQL"; Path = "$NetworkPath\SQL Server 2019 15.0\SQL2019\ExpressAdv_ENU\SETUP.EXE" },
    [PSCustomObject]@{ ID = "sql2022"; Name = "SQL Server 2022 Express"; Type = "Exe"; Path = "$NetworkPath\SQL Server 2022 16.0\SQL2022-SSEI-Expr.exe"; Args = "/Action=Install /IACCEPTSQLSERVERLICENSETERMS /Quiet" },
    [PSCustomObject]@{ ID = "ssms"; Name = "SSMS 20.0"; Type = "Exe"; Path = "$NetworkPath\SSMS\SSMS-Setup-ENU 2024 20.0.exe"; Args = "/Quiet /NoRestart" },
    [PSCustomObject]@{ ID = "mysql"; Name = "MySQL Server 8.0"; Type = "Custom_MySQL"; Path = "$NetworkPath\MySQL installer 8.0.35\mysql-installer-community-8.0.35.0 (Compulsory).msi" },
    [PSCustomObject]@{ ID = "mongodb"; Name = "MongoDB 7.0"; Type = "Msi"; Path = "$NetworkPath\MongoDB 7.0.4\mongodb-windows-x86_64-7.0.4-signed.msi"; Args = "/quiet /norestart ADDLOCAL=`"ServerService,Client`"" },
    [PSCustomObject]@{ ID = "rabbitmq"; Name = "RabbitMQ + Erlang"; Type = "Script"; Path = "$NetworkPath\RabbitMQ\RabbitMQ-FullOfflineInstall.ps1" }
)

# --- FUNCTIONS ---
function Ensure-Choco {
    if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
    }
}

function Install-SQL2019 ($path, $cfgJson) {
    # Default Config
    $InstanceName = "MSSQLSERVER"
    $Features = "SQLEngine,Replication"
    $Password = "triveni@123"
    
    if ($cfgJson) {
        $cfg = $cfgJson | ConvertFrom-Json
        if ($cfg.InstanceName) { $InstanceName = $cfg.InstanceName }
        if ($cfg.Password) { $Password = $cfg.Password }
    }
    
    $Args = "/Q /ACTION=Install /IACCEPTSQLSERVERLICENSETERMS /FEATURES=$Features /INSTANCENAME=$InstanceName /SQLSVCINSTANTFILEINIT=`"True`" /BROWSERSVCSTARTUPTYPE=`"Automatic`" /SECURITYMODE=SQL /SAPWD=`"$Password`" /SQLSYSADMINACCOUNTS=`"BUILTIN\ADMINISTRATORS`""
    Start-Process $path -ArgumentList $Args -Wait
}

function Install-MySQL ($path) {
    Start-Process "msiexec.exe" -ArgumentList "/i `"$path`" /quiet /norestart" -Wait
    $tool = "${env:ProgramFiles(x86)}\MySQL\MySQL Installer for Windows\MySQLInstallerConsole.exe"
    if (!(Test-Path $tool)) { $tool = "${env:ProgramFiles}\MySQL\MySQL Installer for Windows\MySQLInstallerConsole.exe" }
    
    if (Test-Path $tool) {
        Start-Process $tool -ArgumentList "install server;8.0.35;x64 workbench;8.0.34;x64 -silent" -Wait
        Start-Sleep -Seconds 15
        Start-Process $tool -ArgumentList "config server --servicename=MySQL80 --accounts=`"root:triveni@123`" --legacy-auth=true --startuptype=auto -silent" -Wait
    }
}

# --- EXECUTION ---

if ($List) {
    # Force array output to ensure valid JSON array even if 1 item
    @($SoftwareCatalog) | Select-Object ID, Name, Type | ConvertTo-Json -Depth 2 -Compress
    exit
}

if ($Install -and $AppID) {
    $App = $SoftwareCatalog | Where-Object { $_.ID -eq $AppID }
    if (!$App) { 
        Write-Output (@{Status = "Error"; Message = "AppID not found" } | ConvertTo-Json -Compress)
        exit 
    }
    
    $Result = @{ Status = "Success"; Message = "Installed $($App.Name)" }
    
    try {
        switch ($App.Type) {
            "Choco" {
                Ensure-Choco
                choco install $App.Package -y --no-progress | Out-Null
            }
            "Exe" {
                if (Test-Path $App.Path) { Start-Process $App.Path -ArgumentList $App.Args -Wait }
                else { throw "Installer source not found: $($App.Path)" }
            }
            "Msi" {
                if (Test-Path $App.Path) { Start-Process "msiexec.exe" -ArgumentList "/i `"$($App.Path)`" $($App.Args)" -Wait }
                else { throw "Installer source not found" }
            }
            "Custom_SQL" {
                if (Test-Path $App.Path) { Install-SQL2019 $App.Path $Config }
                else { throw "Installer source not found" }
            }
            "Custom_MySQL" {
                if (Test-Path $App.Path) { Install-MySQL $App.Path }
                else { throw "Installer source not found" }
            }
            "Script" {
                if (Test-Path $App.Path) { Start-Process "powershell.exe" -ArgumentList "-ExecutionPolicy Bypass -File `"$($App.Path)`"" -Wait }
                else { throw "Script source not found" }
            }
        }
    }
    catch {
        $Result.Status = "Error"
        $Result.Message = $_.Exception.Message
    }
    
    $Result | ConvertTo-Json -Compress
}
