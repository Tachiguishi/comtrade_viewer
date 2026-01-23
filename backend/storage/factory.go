package storage

import (
	"fmt"

	"comtradeviewer/config"
)

// NewStorage 根据配置创建存储实例
func NewStorage(cfg *config.Config) (Storage, error) {
	switch cfg.Storage.Type {
	case config.StorageTypeLocal:
		return NewLocalStorage(cfg.Storage.Local.BasePath)
	case config.StorageTypeMinIO:
		minIOCfg := MinIOConfig{
			Endpoint:        cfg.Storage.MinIO.Endpoint,
			AccessKeyID:     cfg.Storage.MinIO.AccessKeyID,
			SecretAccessKey: cfg.Storage.MinIO.SecretAccessKey,
			BucketName:      cfg.Storage.MinIO.BucketName,
			UseSSL:          cfg.Storage.MinIO.UseSSL,
		}
		return NewMinIOStorage(minIOCfg)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
