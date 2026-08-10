//go:build !windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WriteUpdaterScript writes a POSIX shell script into exeDir that waits for the
// running process to exit, swaps in the new binary, restarts, then self-deletes.
func (s *UpdateService) WriteUpdaterScript(exeDir, exeName, newName string) (string, error) {
	oldName := strings.TrimSuffix(exeName, filepath.Ext(exeName)) + ".old" + filepath.Ext(exeName)
	script := fmt.Sprintf(`#!/bin/sh
ORIG_CWD=$(pwd)
cd "$(dirname "$0")"
EXE=%s
EXE_ABS="$(pwd)/%s"
NEW=%s
OLD=%s
WAIT=30

echo "[updater] waiting for process to exit..."
i=0
while [ $i -lt $WAIT ]; do
  if ! pgrep -x "%s" >/dev/null 2>&1 && ! pgrep -x "%s" >/dev/null 2>&1; then
    break
  fi
  i=$((i+1))
  sleep 1
done

echo "[updater] installing new version..."
mv -f "$EXE" "$OLD"
mv -f "$NEW" "$EXE"
chmod +x "$EXE"
cd "$ORIG_CWD"
echo "[updater] starting new version..."
nohup "$EXE_ABS" >/dev/null 2>&1 &
rm -f "$0"
`, exeName, exeName, newName, oldName, exeName, exeName)

	path := filepath.Join(exeDir, "kh-e-library-updater.sh")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		return "", err
	}
	return path, nil
}

// LaunchUpdater launches the shell script detached via nohup.
func (s *UpdateService) LaunchUpdater(updaterPath string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("nohup sh %s >/dev/null 2>&1 &", shellQuote(updaterPath)))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("无法启动更新脚本: %w", err)
	}
	return nil
}

// shellQuote wraps a path in single quotes for safe shell interpolation.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
