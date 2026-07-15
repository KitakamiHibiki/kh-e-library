package storage

import (
	"io"
	"time"
)

// FileInfo 包含存储后端中一个文件的元信息
type FileInfo struct {
	Path    string    // 文件在存储后端中的唯一标识
	Name    string    // 原始文件名
	Size    int64     // 文件大小（字节）
	ModTime time.Time // 最后修改时间
	ETag    string    // 可选：校验和 / 版本标识
}

// Driver 定义了书籍文件存储的统一抽象层。
// 所有存储后端（本地文件系统、百度网盘等）均需实现此接口。
type Driver interface {
	// Name 返回存储后端的名称标识，如 "local" / "baidu"
	Name() string

	// Save 将 reader 中的内容以 filename 为名保存到存储后端。
	// 返回存储路径（唯一标识）及实际写入的字节数。
	Save(filename string, reader io.Reader) (path string, size int64, err error)

	// Open 根据 path（Save 返回的标识）打开文件用于读取。
	Open(path string) (io.ReadCloser, error)

	// Delete 根据 path 删除存储后端的文件。
	Delete(path string) error

	// Exists 检查指定 path 的文件是否存在于后端。
	Exists(path string) (bool, error)

	// GetURL 返回文件的直接访问 URL。
	// 本地后端返回文件系统路径，云端后端返回可下载的临时链接。
	// ephemeral 为 true 时表示 URL 有有效期，调用方不应缓存。
	GetURL(path string) (url string, ephemeral bool, err error)

	// Stat 获取文件的详细元信息。
	// 实现方应尽可能填充 FileInfo 的各个字段。
	Stat(path string) (*FileInfo, error)
}

// Factory 管理多个存储后端的注册与获取。
type Factory struct {
	drivers map[string]Driver
}

func NewFactory() *Factory {
	return &Factory{drivers: make(map[string]Driver)}
}

func (f *Factory) Register(name string, d Driver) {
	f.drivers[name] = d
}

func (f *Factory) Get(name string) Driver {
	return f.drivers[name]
}

// Cleanup 释放所有注册的后端资源。
func (f *Factory) Cleanup() {
	for _, d := range f.drivers {
		if c, ok := d.(io.Closer); ok {
			c.Close()
		}
	}
}
