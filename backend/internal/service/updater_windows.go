//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// WriteUpdaterScript writes a Windows batch script into exeDir that:
//  1. waits for the running process to exit
//  2. renames the current exe to .old (kept for manual rollback)
//  3. moves the staged new binary into place
//  4. starts the new version
//  5. self-deletes
//
// Returns the path to the written script.
//
// The script is generated via string replacement (NOT fmt.Sprintf) because
// batch files use %VAR% syntax which fmt would misinterpret as verbs.
func (s *UpdateService) WriteUpdaterScript(exeDir, exeName, newName string) (string, error) {
	tmpl := `@echo off
set "ORIG_CWD=%CD%"
cd /d "%~dp0"
set "EXE=__EXE__"
set "NEW=__NEW__"
set "OLD=__OLD__"
set "WAIT_SECONDS=30"

echo [updater] waiting for process to exit...
set /a waited=0
:waitloop
tasklist /fi "IMAGENAME eq __EXE__" 2>nul | find /i "__EXE__" >nul 2>&1
if errorlevel 1 goto :install
if %waited% geq %WAIT_SECONDS% (
    echo [updater] timed out waiting for __EXE__ to exit.
    del "%~f0" >nul 2>&1
    exit /b 1
)
set /a waited+=1
timeout /t 1 /nobreak >nul
goto :waitloop

:install
echo [updater] installing new version...
move /y "%EXE%" "%OLD%" >nul 2>&1
move /y "%NEW%" "%EXE%" >nul 2>&1
if not exist "%EXE%" (
    echo [updater] failed to install, restoring old version.
    move /y "%OLD%" "%EXE%" >nul 2>&1
    del "%~f0" >nul 2>&1
    exit /b 1
)
echo [updater] starting new version...
start "" /D "%ORIG_CWD%" "%EXE%"
del "%~f0" >nul 2>&1
exit /b 0
`
	script := tmpl
	script = strings.ReplaceAll(script, "__EXE__", exeName)
	script = strings.ReplaceAll(script, "__NEW__", newName)
	script = strings.ReplaceAll(script, "__OLD__", oldName(exeName))

	path := filepath.Join(exeDir, "kh-e-library-updater.bat")
	if err := os.WriteFile(path, []byte(script), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// oldName derives the rollback name for a given exe filename.
func oldName(exeName string) string {
	ext := filepath.Ext(exeName)
	base := exeName[:len(exeName)-len(ext)]
	return base + ".old" + ext
}

// LaunchUpdater launches the updater batch script in a detached console window
// so it survives the main process exiting.
func (s *UpdateService) LaunchUpdater(updaterPath string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", "/min", updaterPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("无法启动更新脚本: %w", err)
	}
	return nil
}
