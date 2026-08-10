package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kitakami-hibiki/e-library/internal/model"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.0.3", "v0.1.0", -1},
		{"v0.1.0", "v0.0.3", 1},
		{"v0.1.0", "v0.1.0", 0},
		{"v0.1.2", "v0.1.10", -1},
		{"1.2.3", "v1.2.3", 0},
		{"v2.0.0", "v10.0.0", -1},
		{"dev", "v0.1.0", -1}, // unparseable treated as 0
		{"v0.1.0", "dev", 1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestExtractZipArchive(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "update.zip")

	// Build a zip containing a nested binary and an unrelated file.
	f, err := os.Create(pkg)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("folder/kh-e-library.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("MZ fake exe")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	svc := NewUpdateService(nil)
	got, err := svc.extractZipArchive(pkg)
	if err != nil {
		t.Fatalf("extractZipArchive failed: %v", err)
	}
	if filepath.Base(got) != "kh-e-library.exe" {
		t.Errorf("extracted %q, want kh-e-library.exe", got)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "MZ fake exe" {
		t.Errorf("content = %q, want %q", string(data), "MZ fake exe")
	}
}

func TestPickAssetPrefersPlatform(t *testing.T) {
	svc := NewUpdateService(nil)
	assets := []model.GitHubAsset{
		{Name: "kh-e-library-v0.1.0-linux-amd64.tar.gz"},
		{Name: "kh-e-library-v0.1.0-windows-amd64.zip"},
		{Name: "kh-e-library-v0.1.0-darwin-arm64.tar.gz"},
	}
	got := svc.pickAsset(assets)
	if got == nil || got.Name != "kh-e-library-v0.1.0-windows-amd64.zip" {
		t.Errorf("pickAsset = %v, want windows-amd64 asset", got)
	}
}

func TestWriteUpdaterScript(t *testing.T) {
	dir := t.TempDir()
	svc := NewUpdateService(nil)
	path, err := svc.WriteUpdaterScript(dir, "kh-e-library.exe", "kh-e-library.new.exe")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// The exe name must be interpolated, and batch %VAR% syntax must be intact.
	for _, want := range []string{
		"set \"EXE=kh-e-library.exe\"",
		"set \"NEW=kh-e-library.new.exe\"",
		"set \"OLD=kh-e-library.old.exe\"",
		"%EXE%",
		"%OLD%",
		"if errorlevel 1 goto :install",
		"start \"\" /D \"%ORIG_CWD%\" \"%EXE%\"",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("updater script missing %q\n---\n%s", want, content)
		}
	}
}
