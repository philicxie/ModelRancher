package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ml-platform/internal/model"
	"ml-platform/internal/repository"
	"ml-platform/pkg/cosclient"

	"github.com/google/uuid"
)

// StorageService 存储服务
type StorageService struct {
	client     *cosclient.Client
	bucketRepo *repository.StorageBucketRepository
}

// NewStorageService 创建存储服务
func NewStorageService(client *cosclient.Client, bucketRepo *repository.StorageBucketRepository) *StorageService {
	return &StorageService{client: client, bucketRepo: bucketRepo}
}

// bucketPrefix 获取桶在COS中的前缀路径
func bucketPrefix(bucketName string) string {
	return "buckets/" + bucketName + "/"
}

// ============================================================================
// 文件对象操作
// ============================================================================

// ListObjects 列出对象
func (s *StorageService) ListObjects(ctx context.Context, bucketName, prefix string) ([]StorageObject, error) {
	if s.client == nil {
		return nil, fmt.Errorf("COS client not initialized")
	}

	fullPrefix := bucketPrefix(bucketName) + prefix
	objects, err := s.client.ListObjects(ctx, fullPrefix)
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

// GetPresignedURL 获取预签名URL
func (s *StorageService) GetPresignedURL(ctx context.Context, bucketName, key string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("COS client not initialized")
	}
	fullKey := bucketPrefix(bucketName) + key
	return s.client.GetPresignedURL(ctx, fullKey, 24*time.Hour)
}

// DeleteObject 删除对象
func (s *StorageService) DeleteObject(ctx context.Context, bucketName, key string) error {
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}
	fullKey := bucketPrefix(bucketName) + key
	return s.client.DeleteObject(ctx, fullKey)
}

// UploadFromReader 从Reader上传文件到COS
func (s *StorageService) UploadFromReader(ctx context.Context, bucketName, key string, reader io.Reader, size int64) error {
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}
	fullKey := bucketPrefix(bucketName) + key
	return s.client.UploadFromReader(ctx, fullKey, reader, size)
}

// CreateFolder 创建文件夹
func (s *StorageService) CreateFolder(ctx context.Context, bucketName, key string) error {
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}
	fullKey := bucketPrefix(bucketName) + key
	return s.client.CreateFolder(ctx, fullKey)
}

// ============================================================================
// 桶管理
// ============================================================================

// ListBuckets 列出用户可见的桶（自己的 + 公共的）
func (s *StorageService) ListBuckets(ctx context.Context, userID string) ([]*model.StorageBucket, error) {
	if s.bucketRepo == nil {
		return nil, fmt.Errorf("bucket repository not initialized")
	}

	// 获取用户自己的桶
	myBuckets, err := s.bucketRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 获取公共桶（排除用户自己的）
	publicBuckets, err := s.bucketRepo.ListPublic(ctx)
	if err != nil {
		return nil, err
	}

	// 合并，去重
	seen := make(map[string]bool)
	result := make([]*model.StorageBucket, 0, len(myBuckets)+len(publicBuckets))

	for _, b := range myBuckets {
		if !seen[b.ID] {
			seen[b.ID] = true
			result = append(result, b)
		}
	}
	for _, b := range publicBuckets {
		if !seen[b.ID] {
			seen[b.ID] = true
			result = append(result, b)
		}
	}

	return result, nil
}

// CreateBucket 创建桶
func (s *StorageService) CreateBucket(ctx context.Context, userID, name string, isPublic bool) (*model.StorageBucket, error) {
	if s.bucketRepo == nil {
		return nil, fmt.Errorf("bucket repository not initialized")
	}
	if s.client == nil {
		return nil, fmt.Errorf("COS client not initialized")
	}

	// 检查桶名是否已存在
	if _, err := s.bucketRepo.GetByName(ctx, name); err == nil {
		return nil, fmt.Errorf("bucket name already exists: %s", name)
	}

	bucket := &model.StorageBucket{
		ID:        uuid.New().String(),
		Name:      name,
		UserID:    userID,
		IsPublic:  isPublic,
		CreatedAt: time.Now(),
	}

	// 写入数据库
	if err := s.bucketRepo.Create(ctx, bucket); err != nil {
		return nil, err
	}

	// 在COS中创建桶的根文件夹
	if err := s.client.CreateFolder(ctx, bucketPrefix(name)); err != nil {
		log.Printf("[storage] Warning: failed to create COS folder for bucket %s: %v", name, err)
	}

	return bucket, nil
}

// DeleteBucket 删除桶
func (s *StorageService) DeleteBucket(ctx context.Context, bucketID, userID string) error {
	if s.bucketRepo == nil {
		return fmt.Errorf("bucket repository not initialized")
	}
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}

	bucket, err := s.bucketRepo.GetByID(ctx, bucketID)
	if err != nil {
		return err
	}

	// 只有桶所有者可以删除
	if bucket.UserID != userID {
		return fmt.Errorf("permission denied: only bucket owner can delete")
	}

	// 删除COS中该桶前缀下所有对象
	prefix := bucketPrefix(bucket.Name)
	if err := s.client.DeleteObjectsByPrefix(ctx, prefix); err != nil {
		log.Printf("[storage] Warning: failed to delete COS objects for bucket %s: %v", bucket.Name, err)
	}

	// 删除数据库记录
	return s.bucketRepo.Delete(ctx, bucketID)
}

// GetBucketByID 根据ID获取桶
func (s *StorageService) GetBucketByID(ctx context.Context, bucketID string) (*model.StorageBucket, error) {
	if s.bucketRepo == nil {
		return nil, fmt.Errorf("bucket repository not initialized")
	}
	return s.bucketRepo.GetByID(ctx, bucketID)
}

// ============================================================================
// 其他
// ============================================================================

// DownloadData 下载数据
func (s *StorageService) DownloadData(ctx context.Context, cosPath, localPath string) error {
	if s.client == nil {
		return fmt.Errorf("COS client not initialized")
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}
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
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
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

// PresignedDownload 预签名下载URL条目
type PresignedDownload struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// GenerateDownloadURLs 为指定前缀下的所有对象生成预签名下载URL
func (s *StorageService) GenerateDownloadURLs(ctx context.Context, bucketName, prefix string) ([]PresignedDownload, error) {
	if s.client == nil {
		return nil, fmt.Errorf("COS client not initialized")
	}

	objects, err := s.ListObjects(ctx, bucketName, prefix)
	if err != nil {
		return nil, err
	}

	result := make([]PresignedDownload, 0, len(objects))
	for _, obj := range objects {
		if obj.IsDir {
			continue
		}
		relKey := strings.TrimPrefix(obj.Key, bucketPrefix(bucketName))
		url, err := s.GetPresignedURL(ctx, bucketName, relKey)
		if err != nil {
			log.Printf("failed to generate presigned URL for %s: %v", obj.Key, err)
			continue
		}
		result = append(result, PresignedDownload{
			Key: relKey,
			URL: url,
		})
	}
	return result, nil
}

// GenerateUploadURL 为单个对象生成预签名上传URL（PUT）
func (s *StorageService) GenerateUploadURL(ctx context.Context, bucketName, key string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("COS client not initialized")
	}
	fullKey := bucketPrefix(bucketName) + key
	return s.client.GetPresignedURL(ctx, fullKey, 24*time.Hour)
}

// StorageObject 存储对象
type StorageObject struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
	IsDir        bool   `json:"isDir"`
}
