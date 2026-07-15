package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalDriver struct {
	BooksDir string
}

func NewLocalDriver(booksDir string) *LocalDriver {
	if err := os.MkdirAll(booksDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create books directory: %v", err))
	}
	return &LocalDriver{BooksDir: booksDir}
}

func (d *LocalDriver) Name() string {
	return "local"
}

func (d *LocalDriver) Save(filename string, reader io.Reader) (string, int64, error) {
	safeName := filepath.Base(filename)
	dest := filepath.Join(d.BooksDir, safeName)

	if _, err := os.Stat(dest); err == nil {
		ext := filepath.Ext(safeName)
		base := strings.TrimSuffix(safeName, ext)
		for i := 1; ; i++ {
			dest = filepath.Join(d.BooksDir, fmt.Sprintf("%s_%d%s", base, i, ext))
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				break
			}
		}
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, reader)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write file: %w", err)
	}

	return dest, written, nil
}

func (d *LocalDriver) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (d *LocalDriver) Delete(path string) error {
	return os.Remove(path)
}

func (d *LocalDriver) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (d *LocalDriver) GetURL(path string) (string, bool, error) {
	// 本地后端直接返回文件系统路径，不涉及有效期
	return path, false, nil
}

func (d *LocalDriver) Stat(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Path:    path,
		Name:    filepath.Base(path),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

