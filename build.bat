@echo off
setlocal enabledelayedexpansion

REM 项目版本号统一由环境变量 E_LIBRARY_VERSION 提供，未设置时回退到 VERSION 文件。
if not defined E_LIBRARY_VERSION (
  for /f "usebackq delims=" %%a in ("%~dp0VERSION") do set E_LIBRARY_VERSION=%%a
)

echo === Building frontend ===
cd /d "%~dp0client\web"
call npm.cmd install
call npm.cmd run build

echo === Copying frontend dist to backend ===
if exist "%~dp0backend\web\dist" rmdir /S /Q "%~dp0backend\web\dist"
mkdir "%~dp0backend\web\dist"
xcopy /E /Y "%~dp0client\web\dist" "%~dp0backend\web\dist\"
REM Keep the tracked go:embed placeholder so a fresh checkout still compiles.
type nul > "%~dp0backend\web\dist\.gitkeep"

echo === Building backend ===
cd /d "%~dp0backend"
set CGO_ENABLED=0
go build -ldflags="-s -w -X main.Version=%E_LIBRARY_VERSION%" -o kh-e-library.exe ./cmd/server

echo === Done! Binary: backend\kh-e-library.exe
pause


