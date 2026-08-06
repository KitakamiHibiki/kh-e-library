package storage

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// LocalDriver implements storage.Driver using the local filesystem.
// Files are organized as {BooksDir}/{bookID}/{fileName}.
type LocalDriver struct {
	BooksDir string
}

// NewLocalDriver creates a LocalDriver and ensures the base directory exists.
func NewLocalDriver(booksDir string) (*LocalDriver, error) {
	if err := os.MkdirAll(booksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create books directory: %w", err)
	}
	return &LocalDriver{BooksDir: booksDir}, nil
}

func (d *LocalDriver) Name() string { return "local" }

// Save stores a file under the book's directory.
// The filename is sanitized via filepath.Base() to prevent path traversal.
// Returns the sanitized filename and the number of bytes written.
func (d *LocalDriver) Save(bookID uint, filename string, reader io.Reader) (string, int64, error) {
	safeName, err := sanitizeFilename(filename)
	if err != nil {
		return "", 0, err
	}

	bookDir := d.bookDir(bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create book directory: %w", err)
	}

	dest := filepath.Join(bookDir, safeName)
	f, err := os.Create(dest)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, reader)
	if err != nil {
		// Clean up partially written file
		os.Remove(dest)
		return "", 0, fmt.Errorf("failed to write file: %w", err)
	}

	return safeName, written, nil
}

// Open opens a file for reading by bookID and fileName.
// Returns *os.File which implements ReadSeekCloser for efficient Range request handling.
func (d *LocalDriver) Open(bookID uint, fileName string) (ReadSeekCloser, error) {
	safeName, err := sanitizeFilename(fileName)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(d.bookDir(bookID), safeName))
	if err != nil {
		return nil, err
	}
	return f, nil
}

// DeleteBook removes the entire directory for a book.
func (d *LocalDriver) DeleteBook(bookID uint) error {
	return os.RemoveAll(d.bookDir(bookID))
}

// Delete removes a single file from the book's directory.
func (d *LocalDriver) Delete(bookID uint, fileName string) error {
	safeName, err := sanitizeFilename(fileName)
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(d.bookDir(bookID), safeName))
}

// Exists checks whether a file exists in the book's directory.
func (d *LocalDriver) Exists(bookID uint, fileName string) (bool, error) {
	safeName, err := sanitizeFilename(fileName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(d.bookDir(bookID), safeName))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// GetURL returns a relative URL path for the file. Not ephemeral.
// Uses forward slashes for URL compatibility regardless of OS.
func (d *LocalDriver) GetURL(bookID uint, fileName string) (string, bool, error) {
	safeName, err := sanitizeFilename(fileName)
	if err != nil {
		return "", false, err
	}
	return path.Join(strconv.Itoa(int(bookID)), safeName), false, nil
}

// Stat returns file metadata.
func (d *LocalDriver) Stat(bookID uint, fileName string) (*FileInfo, error) {
	safeName, err := sanitizeFilename(fileName)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(filepath.Join(d.bookDir(bookID), safeName))
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Path:    path.Join(strconv.Itoa(int(bookID)), safeName),
		Name:    safeName,
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
	}, nil
}

// bookDir returns the directory path for a given bookID.
func (d *LocalDriver) bookDir(bookID uint) string {
	return filepath.Join(d.BooksDir, strconv.Itoa(int(bookID)))
}

// sanitizeFilename cleans a filename to prevent path traversal attacks.
// It applies filepath.Base() and then rejects names containing path separators,
// drive letters, or starting with a dot.
func sanitizeFilename(name string) (string, error) {
	safe := filepath.Base(name)
	if safe == "" || safe == "." || safe == ".." {
		return "", fmt.Errorf("invalid filename: %q", name)
	}
	if strings.ContainsAny(safe, `\/:`) {
		return "", fmt.Errorf("filename contains invalid characters: %q", safe)
	}
	if strings.HasPrefix(safe, ".") {
		return "", fmt.Errorf("filename must not start with a dot: %q", safe)
	}
	return safe, nil
}
