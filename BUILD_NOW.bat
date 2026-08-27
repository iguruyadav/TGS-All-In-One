@echo off
echo ============================================
echo   TGS ALL-IN-ONE :: REBUILDING APP
echo ============================================
cd /d "c:\Users\TGSUser012\Desktop\MY Software\TGS_All_In_One"

echo.
echo [1/2] Generating Wails bindings (new StopInstall method)...
wails generate module
if %errorlevel% neq 0 (
    echo WARNING: generate module had issues, continuing with build...
)

echo.
echo [2/2] Building application...
wails build
if %errorlevel% neq 0 (
    echo.
    echo BUILD FAILED! See error above.
    pause
    exit /b 1
)

echo.
echo ============================================
echo   BUILD COMPLETE!
echo   New EXE: build\bin\TGS_All_In_One.exe
echo ============================================
pause
