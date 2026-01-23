package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	basePath string
}

// NewLocalStorage 创建本地存储实例
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// 确保基础路径存在
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// SaveFile 保存文件到本地存储
func (ls *LocalStorage) SaveFile(ctx context.Context, path string, data io.Reader) error {
	// 构建完整路径
	fullPath := filepath.Join(ls.basePath, path)

	// 创建必要的目录
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 写入数据
	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ReadFile 从本地存储读取文件
func (ls *LocalStorage) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(ls.basePath, path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// FileExists 检查文件是否存在
func (ls *LocalStorage) FileExists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(ls.basePath, path)

	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// DeleteFile 删除文件
func (ls *LocalStorage) DeleteFile(ctx context.Context, path string) error {
	fullPath := filepath.Join(ls.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在时不报错
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// ListFiles 列出指定目录下的所有文件
func (ls *LocalStorage) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	prefixPath := filepath.Join(ls.basePath, prefix)

	var files []string
	err := filepath.Walk(prefixPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// 返回相对于basePath的路径
			relPath, _ := filepath.Rel(ls.basePath, path)
			files = append(files, relPath)
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}

// GetFileSize 获取文件大小
func (ls *LocalStorage) GetFileSize(ctx context.Context, path string) (int64, error) {
	fullPath := filepath.Join(ls.basePath, path)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file size: %w", err)
	}

	return stat.Size(), nil
}

// Close 关闭本地存储（无需实际操作）
func (ls *LocalStorage) Close() error {
	return nil
}
