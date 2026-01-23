package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage MinIO对象存储实现
type MinIOStorage struct {
	client     *minio.Client
	bucketName string
}

// MinIOConfig MinIO配置信息
type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

// NewMinIOStorage 创建MinIO存储实例
func NewMinIOStorage(cfg MinIOConfig) (*MinIOStorage, error) {
	// 初始化MinIO客户端
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// 检查bucket是否存在，不存在则创建
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

// SaveFile 保存文件到MinIO存储
func (ms *MinIOStorage) SaveFile(ctx context.Context, path string, data io.Reader) error {
	// 获取数据长度（需要reader支持Seek）
	// 对于大文件，使用-1表示使用流式上传
	_, err := ms.client.PutObject(ctx, ms.bucketName, path, data, -1, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// ReadFile 从MinIO存储读取文件
func (ms *MinIOStorage) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	object, err := ms.client.GetObject(ctx, ms.bucketName, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	// 检查是否真的存在
	_, err = object.Stat()
	if err != nil {
		return nil, fmt.Errorf("object not found: %w", err)
	}

	return object, nil
}

// FileExists 检查文件是否存在
func (ms *MinIOStorage) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := ms.client.StatObject(ctx, ms.bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object: %w", err)
	}

	return true, nil
}

// DeleteFile 删除文件
func (ms *MinIOStorage) DeleteFile(ctx context.Context, path string) error {
	err := ms.client.RemoveObject(ctx, ms.bucketName, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// ListFiles 列出指定目录下的所有文件
func (ms *MinIOStorage) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range ms.client.ListObjects(ctx, ms.bucketName, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		// 过滤掉以/结尾的目录对象
		if !strings.HasSuffix(object.Key, "/") {
			files = append(files, object.Key)
		}
	}

	return files, nil
}

// GetFileSize 获取文件大小
func (ms *MinIOStorage) GetFileSize(ctx context.Context, path string) (int64, error) {
	stat, err := ms.client.StatObject(ctx, ms.bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to stat object: %w", err)
	}

	return stat.Size, nil
}

// Close 关闭MinIO存储连接
func (ms *MinIOStorage) Close() error {
	return nil
}
