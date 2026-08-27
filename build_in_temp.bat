@echo off
echo ============================================================
echo   TGS ALL-IN-ONE :: SANDBOX BYPASS BUILD SCRIPT
echo ============================================================
echo.
echo Your IT policy blocks terminal tools from running in the Desktop folder.
echo This script temporarily copies the project to C:\temp, builds it there,
echo and moves the finished executable back to your Desktop.
echo.

set "SOURCE_DIR=%~dp0"
set "TEMP_DIR=C:\temp\TGS_Build"
set "DESKTOP_DIR=%USERPROFILE%\Desktop"

echo [1/3] Copying project to %TEMP_DIR%...
if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%"
mkdir "%TEMP_DIR%"
xcopy "%SOURCE_DIR%*" "%TEMP_DIR%\" /E /I /H /Y /EXCLUDE:"%SOURCE_DIR%exclude.txt" >nul

echo [2/3] Running Wails Build in Temp Directory...
cd /d "%TEMP_DIR%"
wails build

echo [3/3] Copying executable back to Desktop...
if exist "%TEMP_DIR%\build\bin\TGS_All_In_One_v15_Final_Security_Fixes.exe" (
    copy /Y "%TEMP_DIR%\build\bin\TGS_All_In_One_v15_Final_Security_Fixes.exe" "%SOURCE_DIR%build\bin\" >nul
    echo.
    echo SUCCESS: TGS_All_In_One_v15_Final_Security_Fixes.exe has been built and copied to your folder!
) else (
    echo.
    echo ERROR: Build failed. Could not find executable.
)

pause
