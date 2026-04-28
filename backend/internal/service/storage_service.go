package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ml-platform/pkg/cosclient"
)

// StorageService 存储服务
type StorageService struct {
	client *cosclient.Client
}

// NewStorageService 创建存储服务
func NewStorageService(client *cosclient.Client) *StorageService {
	return &StorageService{client: client}
}

// ListObjects 列出对象
func (s *StorageService) ListObjects(ctx context.Context, prefix string) ([]StorageObject, error) {
	if s.client == nil {
		return nil, fmt.Errorf("COS client not initialized")
	}

	objects, err := s.client.ListObjects(ctx, prefix)
	if err != nil {
		return nil, err
	}

	result := make([]StorageObject, 0, len(objects))
	for _, obj := range objects {
		result = append(result, StorageObject{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			IsDir:        strings.HasSuffix(obj.Key, "/"),
		})
	}

	return result, nil
}

// DownloadData 下载数据
func (s *StorageService) DownloadData(ctx context.Context, cosPath, localPath string) error {
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}

	// 确保本地目录存在
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// 下载对象
	if err := s.client.DownloadObject(ctx, cosPath, localPath); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	return nil
}

// UploadResults 上传结果
func (s *StorageService) UploadResults(ctx context.Context, taskID, outputPath, localDir string) error {
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}

	// 遍历本地目录
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		cosKey := fmt.Sprintf("%s/%s/%s", outputPath, taskID, relPath)

		if err := s.client.UploadObject(ctx, path, cosKey); err != nil {
			return fmt.Errorf("failed to upload %s: %w", relPath, err)
		}

		return nil
	})
}

// GetPresignedURL 获取预签名URL
func (s *StorageService) GetPresignedURL(ctx context.Context, key string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("COS client not initialized")
	}

	return s.client.GetPresignedURL(ctx, key, 24*time.Hour)
}

// StorageObject 存储对象
type StorageObject struct {
	Key          string
	Size         int64
	LastModified string
	IsDir        bool
}
