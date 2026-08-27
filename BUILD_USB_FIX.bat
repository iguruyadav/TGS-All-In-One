@echo off
title TGS ALL-IN-ONE :: USB FIX BUILD
echo ============================================================
echo   TGS ALL-IN-ONE :: USB FIX BUILD v14.2
echo ============================================================
cd /d "c:\Users\TGSUser012\Desktop\MY Software\TGS_All_In_One"

echo.
echo [1/2] Building application...
wails build
if %errorlevel% neq 0 (
    echo.
    echo ========================================
    echo  BUILD FAILED! See error above.
    echo ========================================
    pause
    exit /b 1
)

echo.
echo [2/2] Renaming output to v14_2_USB_Fix...
if exist "build\bin\TGS_All_In_One_v14_2_USB_Fix.exe" del /f "build\bin\TGS_All_In_One_v14_2_USB_Fix.exe"
if exist "build\bin\TGS_All_In_One_v14_2_USB_Fix.exe" goto skip_rename
copy /y "build\bin\TGS_All_In_One_v14_2_USB_Fix.exe" "build\bin\TGS_All_In_One_v14_2_USB_Fix.exe" >nul 2>&1
:skip_rename

echo.
echo ============================================================
echo   BUILD COMPLETE!
echo   EXE is in: build\bin\
echo ============================================================
echo.

:: Open the bin folder automatically
explorer "build\bin"

pause
