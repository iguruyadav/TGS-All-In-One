@echo off
echo ============================================================
echo  TGS ALL-IN-ONE :: FINAL SECURE BUILD
echo ============================================================
cd /d "c:\Users\TGSUser012\Desktop\MY Software\TGS_All_In_One"

echo.
echo [1/2] Generating Wails bindings...
wails generate module 2>nul
echo (bindings done)

echo.
echo [2/2] Building application...
wails build 2>&1
if %errorlevel% neq 0 (
    echo.
    echo ========================================
    echo  BUILD FAILED! Saving error log...
    echo ========================================
    wails build 2> "%USERPROFILE%\Desktop\TGS_BUILD_ERRORS.txt"
    echo Error saved to Desktop: TGS_BUILD_ERRORS.txt
    echo Please share that file.
    pause
    exit /b 1
)

echo.
echo ============================================================
echo  BUILD COMPLETE!
echo  New EXE: build\bin\TGS_All_In_One_v15_Final_Security_Fixes.exe
echo ============================================================
pause
