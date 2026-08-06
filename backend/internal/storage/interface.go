package storage

import (
	"io"
)

// ReadSeekCloser combines io.Reader, io.Seeker, and io.Closer.
// *os.File implements this interface, allowing efficient Range request handling.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// FileInfo contains metadata about a file in a storage backend.
type FileInfo struct {
	Path    string // unique identifier within the storage backend
	Name    string // original filename
	Size    int64  // file size in bytes
	ModTime int64  // last modified time (Unix seconds)
	ETag    string // optional: checksum / version identifier
}

// Driver defines the unified abstraction for book file storage.
// All methods take bookID as the first parameter; the driver internally
// constructs the full path as {baseDir}/{bookID}/{fileName}.
type Driver interface {
	// Name returns the storage backend identifier, e.g. "local".
	Name() string

	// Save stores the content from reader as the given filename under the book's directory.
	// Returns the saved filename and the number of bytes written.
	Save(bookID uint, filename string, reader io.Reader) (fileName string, size int64, err error)

	// Open opens a file for reading by bookID and fileName.
	// Implementations should return a ReadSeekCloser when possible to support Range requests.
	Open(bookID uint, fileName string) (ReadSeekCloser, error)

	// DeleteBook removes the entire directory for a book (including all files).
	DeleteBook(bookID uint) error

	// Delete removes a single file from the book's directory.
	Delete(bookID uint, fileName string) error

	// Exists checks whether a file exists in the book's directory.
	Exists(bookID uint, fileName string) (bool, error)

	// GetURL returns a URL for direct file access.
	// ephemeral=true means the URL has a limited lifetime and should not be cached.
	GetURL(bookID uint, fileName string) (url string, ephemeral bool, err error)

	// Stat returns file metadata.
	Stat(bookID uint, fileName string) (*FileInfo, error)
}

// Factory manages multiple storage backend drivers by name.
type Factory struct {
	drivers map[string]Driver
}

// NewFactory creates an empty driver factory.
func NewFactory() *Factory {
	return &Factory{drivers: make(map[string]Driver)}
}

// Register adds a driver to the factory.
func (f *Factory) Register(name string, d Driver) {
	f.drivers[name] = d
}

// Get retrieves a driver by name. Returns nil if not found.
func (f *Factory) Get(name string) Driver {
	return f.drivers[name]
}

// Cleanup releases resources held by all registered drivers.
func (f *Factory) Cleanup() {
	for _, d := range f.drivers {
		if c, ok := d.(io.Closer); ok {
			c.Close()
		}
	}
}
