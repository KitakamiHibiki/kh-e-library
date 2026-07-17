@echo off
setlocal enabledelayedexpansion

for /f "usebackq delims=" %%a in ("%~dp0VERSION") do set VERSION=%%a

echo === Building frontend ===
cd /d "%~dp0client\web"
call npm.cmd install
call npm.cmd run build

echo === Copying frontend dist to backend ===
if exist "%~dp0backend\web\dist" rmdir /S /Q "%~dp0backend\web\dist"
mkdir "%~dp0backend\web\dist"
xcopy /E /Y "%~dp0client\web\dist" "%~dp0backend\web\dist\"

echo === Building backend ===
cd /d "%~dp0backend"
go build -ldflags="-s -w -X main.Version=%VERSION%" -o kh-e-library.exe ./cmd/server

echo === Done! Binary: backend\e-library.exe
pause

