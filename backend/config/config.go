package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal  StorageType = "local"
	StorageTypeMinIO  StorageType = "minio"
)

// Config 应用配置
type Config struct {
	Storage StorageConfig `yaml:"storage"`
	Server  ServerConfig  `yaml:"server"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type  StorageType `yaml:"type"`
	Local LocalConfig `yaml:"local"`
	MinIO MinIOConfig `yaml:"minio"`
}

// LocalConfig 本地存储配置
type LocalConfig struct {
	BasePath string `yaml:"basePath"`
}

// MinIOConfig MinIO存储配置
type MinIOConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	BucketName      string `yaml:"bucketName"`
	UseSSL          bool   `yaml:"useSSL"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，使用默认配置
	cfg := &Config{
		Storage: StorageConfig{
			Type: StorageTypeLocal,
			Local: LocalConfig{
				BasePath: "./data",
			},
		},
		Server: ServerConfig{
			Port: 8080,
		},
	}

	// 读取配置文件
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 从环境变量覆盖配置
	cfg.overrideFromEnv()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// overrideFromEnv 从环境变量覆盖配置
func (c *Config) overrideFromEnv() {
	// 存储类型
	if storageType := os.Getenv("STORAGE_TYPE"); storageType != "" {
		c.Storage.Type = StorageType(strings.ToLower(storageType))
	}

	// 本地存储配置
	if basePath := os.Getenv("STORAGE_LOCAL_PATH"); basePath != "" {
		c.Storage.Local.BasePath = basePath
	}

	// MinIO配置
	if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		c.Storage.MinIO.Endpoint = endpoint
	}
	if accessKey := os.Getenv("MINIO_ACCESS_KEY"); accessKey != "" {
		c.Storage.MinIO.AccessKeyID = accessKey
	}
	if secretKey := os.Getenv("MINIO_SECRET_KEY"); secretKey != "" {
		c.Storage.MinIO.SecretAccessKey = secretKey
	}
	if bucketName := os.Getenv("MINIO_BUCKET"); bucketName != "" {
		c.Storage.MinIO.BucketName = bucketName
	}
	if useSSL := os.Getenv("MINIO_USE_SSL"); useSSL != "" {
		c.Storage.MinIO.UseSSL = strings.ToLower(useSSL) == "true"
	}

	// 服务器配置
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	switch c.Storage.Type {
	case StorageTypeLocal:
		if c.Storage.Local.BasePath == "" {
			return fmt.Errorf("local storage basePath is required")
		}
	case StorageTypeMinIO:
		if c.Storage.MinIO.Endpoint == "" {
			return fmt.Errorf("minio endpoint is required")
		}
		if c.Storage.MinIO.AccessKeyID == "" {
			return fmt.Errorf("minio accessKeyId is required")
		}
		if c.Storage.MinIO.SecretAccessKey == "" {
			return fmt.Errorf("minio secretAccessKey is required")
		}
		if c.Storage.MinIO.BucketName == "" {
			return fmt.Errorf("minio bucketName is required")
		}
	default:
		return fmt.Errorf("invalid storage type: %s", c.Storage.Type)
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	return nil
}
