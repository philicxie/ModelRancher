package cosclient

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// Client COS客户端封装
type Client struct {
	client *cos.Client
	bucket string
}

// Config COS配置
type Config struct {
	SecretID  string
	SecretKey string
	Bucket    string
	Region    string
}

// NewCOSClient 创建COS客户端
func NewCOSClient(secretID, secretKey, bucket, region string) (*Client, error) {
	if secretID == "" || secretKey == "" || bucket == "" || region == "" {
		return nil, fmt.Errorf("COS configuration is incomplete")
	}

	u, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	if err != nil {
		return nil, fmt.Errorf("failed to parse COS URL: %w", err)
	}

	client := cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	return &Client{client: client, bucket: bucket}, nil
}

// ListObjects 列出对象
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	opt := &cos.BucketGetOptions{
		Prefix:  prefix,
		MaxKeys: 1000,
	}

	resp, _, err := c.client.Bucket.Get(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	objects := make([]ObjectInfo, 0, len(resp.Contents))
	for _, item := range resp.Contents {
		objects = append(objects, ObjectInfo{
			Key:          item.Key,
			Size:         item.Size,
			LastModified: item.LastModified,
		})
	}

	return objects, nil
}

// DownloadObject 下载对象到本地文件
func (c *Client) DownloadObject(ctx context.Context, key, localPath string) error {
	_, err := c.client.Object.GetToFile(ctx, key, localPath, nil)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", key, err)
	}
	return nil
}

// UploadObject 上传本地文件到对象
func (c *Client) UploadObject(ctx context.Context, localPath, key string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", localPath, err)
	}
	defer f.Close()

	_, err = c.client.Object.Put(ctx, key, f, nil)
	if err != nil {
		return fmt.Errorf("failed to upload %s: %w", key, err)
	}
	return nil
}

// UploadFromReader 从Reader上传
func (c *Client) UploadFromReader(ctx context.Context, key string, reader io.Reader, size int64) error {
	opt := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
			ContentLength: size,
		},
	}
	_, err := c.client.Object.Put(ctx, key, reader, opt)
	if err != nil {
		return fmt.Errorf("failed to upload %s: %w", key, err)
	}
	return nil
}

// CreateFolder 创建文件夹（上传一个空对象，key以/结尾）
func (c *Client) CreateFolder(ctx context.Context, key string) error {
	_, err := c.client.Object.Put(ctx, key, strings.NewReader(""), nil)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", key, err)
	}
	return nil
}

// DeleteObjectsByPrefix 删除指定前缀下所有对象
func (c *Client) DeleteObjectsByPrefix(ctx context.Context, prefix string) error {
	// 列出该前缀下所有对象
	objects, err := c.ListObjects(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list objects for delete: %w", err)
	}

	// 逐个删除
	for _, obj := range objects {
		if err := c.DeleteObject(ctx, obj.Key); err != nil {
			log.Printf("[cos] Failed to delete %s: %v", obj.Key, err)
		}
	}
	return nil
}

// GetPresignedURL 获取预签名下载URL
func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// 参数：ctx, method, key, ak, sk, expiry, opt
	// ak/sk 传空串表示使用 client 初始化时的默认密钥
	u, err := c.client.Object.GetPresignedURL(ctx, http.MethodGet, key, "", "", expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return u.String(), nil
}

// DeleteObject 删除对象
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.Object.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", key, err)
	}
	return nil
}

// ObjectInfo 对象信息
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified string
}
