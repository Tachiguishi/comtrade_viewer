package storage

import (
	"context"
	"io"
)

// Storage 定义文件存储的通用接口
type Storage interface {
	// SaveFile 保存文件到存储
	SaveFile(ctx context.Context, path string, data io.Reader) error

	// ReadFile 从存储读取文件
	ReadFile(ctx context.Context, path string) (io.ReadCloser, error)

	// FileExists 检查文件是否存在
	FileExists(ctx context.Context, path string) (bool, error)

	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, path string) error

	// ListFiles 列出指定目录下的所有文件
	ListFiles(ctx context.Context, prefix string) ([]string, error)

	// GetFileSize 获取文件大小
	GetFileSize(ctx context.Context, path string) (int64, error)

	// Close 关闭存储连接
	Close() error
}
