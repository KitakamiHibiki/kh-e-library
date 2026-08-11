package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// ChunkManager persists and assembles upload chunks in a temporary staging
// area. It lives independently of the Driver abstraction: chunks are stored
// under {BooksDir}/.chunks/{bookID}/ until the final chunk arrives, at which
// point AssembleAndHash merges them and the caller saves the result to the
// real book directory via the storage Driver.
type ChunkManager struct {
	BooksDir string
}

// NewChunkManager creates a ChunkManager rooted at the given books directory.
func NewChunkManager(booksDir string) *ChunkManager {
	return &ChunkManager{BooksDir: booksDir}
}

// chunksDir returns the staging directory for a given bookID.
func (m *ChunkManager) chunksDir(bookID uint) string {
	return filepath.Join(m.BooksDir, ".chunks", strconv.Itoa(int(bookID)))
}

// chunkPath returns the file path of a single chunk.
func (m *ChunkManager) chunkPath(bookID uint, chunkIndex int) string {
	return filepath.Join(m.chunksDir(bookID), fmt.Sprintf("chunk_%d", chunkIndex))
}

// CreateDir ensures the staging directory for a book exists, so a client can
// start pushing chunks right after init.
func (m *ChunkManager) CreateDir(bookID uint) error {
	return os.MkdirAll(m.chunksDir(bookID), 0755)
}

// SaveChunk writes one chunk to the staging directory. Re-writing the same
// index (e.g. a client retry) truncates the previous chunk.
func (m *ChunkManager) SaveChunk(bookID uint, chunkIndex int, reader io.Reader) error {
	dir := m.chunksDir(bookID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %w", err)
	}
	dest := m.chunkPath(bookID, chunkIndex)
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to write chunk file: %w", err)
	}
	return nil
}

// AssembleAndHash merges all chunks (chunk_0 .. chunk_{totalChunks-1}) into a
// single file `assembled.{ext}` inside the staging directory, computing a
// streaming SHA-256 over the combined bytes. It returns the assembled file
// path, the hex-encoded hash, and the total size in bytes.
func (m *ChunkManager) AssembleAndHash(bookID uint, totalChunks int, destExt string) (string, string, int64, error) {
	dir := m.chunksDir(bookID)
	assembledPath := filepath.Join(dir, "assembled"+destExt)

	out, err := os.Create(assembledPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create assembled file: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	var total int64
	for i := 0; i < totalChunks; i++ {
		chunk, err := os.Open(m.chunkPath(bookID, i))
		if err != nil {
			os.Remove(assembledPath)
			return "", "", 0, fmt.Errorf("failed to open chunk %d: %w", i, err)
		}
		n, err := io.Copy(writer, chunk)
		chunk.Close()
		if err != nil {
			os.Remove(assembledPath)
			return "", "", 0, fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}
		total += n
	}

	return assembledPath, hex.EncodeToString(hasher.Sum(nil)), total, nil
}

// Cleanup removes the entire staging directory for a book. It is safe to call
// when the directory does not exist.
func (m *ChunkManager) Cleanup(bookID uint) error {
	return os.RemoveAll(m.chunksDir(bookID))
}
