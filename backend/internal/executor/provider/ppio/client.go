package ppio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	BaseURL = "https://api.ppio.com/gpu-instance/openapi/v1"
)

// Client PPIO API客户端
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// Config PPIO客户端配置
type Config struct {
	APIKey string
}

// NewClient 创建PPIO客户端
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &Client{
		apiKey: cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// doRequest 发送HTTP请求
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ============================================================================
// 产品相关 API
// ============================================================================

// FloatOrString 兼容 API 返回 string 或 number 的价格字段
type FloatOrString float64

func (f *FloatOrString) UnmarshalJSON(data []byte) error {
	// 先尝试解析为 string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v, parseErr := strconv.ParseFloat(s, 64)
		if parseErr != nil {
			return parseErr
		}
		*f = FloatOrString(v)
		return nil
	}
	// 再尝试解析为 float64
	var v float64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*f = FloatOrString(v)
	return nil
}

func (f FloatOrString) Float64() float64 {
	return float64(f)
}

// Product GPU产品信息
type Product struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	CpuPerGpu       int           `json:"cpuPerGpu"`
	MemoryPerGpu    int           `json:"memoryPerGpu"`
	DiskPerGpu      int           `json:"diskPerGpu"`
	AvailableDeploy bool          `json:"availableDeploy"`
	MinRootFS       int           `json:"minRootFS"`
	MaxRootFS       int           `json:"maxRootFS"`
	Regions         []string      `json:"regions"`
	Price           FloatOrString `json:"price"`
	SpotPrice       string        `json:"spotPrice"`
	InventoryState  string        `json:"inventoryState"`
	BillingMethods  []string      `json:"billingMethods"`
}

// ListProductsResponse 产品列表响应
type ListProductsResponse struct {
	Data []Product `json:"data"`
}

// ListProducts 获取GPU产品列表
func (c *Client) ListProducts(ctx context.Context, gpuType string, gpuNum int) ([]Product, error) {
	q := url.Values{}
	q.Set("type", "gpu")
	if gpuType != "" {
		q.Set("name", gpuType)
	}
	if gpuNum > 0 {
		q.Set("gpuNum", fmt.Sprintf("%d", gpuNum))
	}

	respBody, err := c.doRequest(ctx, "GET", "/products?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListProductsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return result.Data, nil
}

// ============================================================================
// 实例相关 API
// ============================================================================

// Env 环境变量
type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CreateGPUInstanceRequest 创建GPU实例请求
type CreateGPUInstanceRequest struct {
	Name           string `json:"name"`
	ProductID      string `json:"productId"`
	GpuNum         int    `json:"gpuNum"`
	RootfsSize     int    `json:"rootfsSize"`
	ImageUrl       string `json:"imageUrl"`
	ImageAuth      string `json:"imageAuth,omitempty"`
	ImageAuthId    string `json:"imageAuthId,omitempty"`
	Ports          string `json:"ports,omitempty"`
	Envs           []Env  `json:"envs,omitempty"`
	Command        string `json:"command,omitempty"`
	Entrypoint     string `json:"entrypoint,omitempty"`
	ClusterId      string `json:"clusterId,omitempty"`
	NetworkId      string `json:"networkId,omitempty"`
	Kind           string `json:"kind,omitempty"`
	BillingMode    string `json:"billingMode,omitempty"`
	MinCudaVersion string `json:"minCudaVersion,omitempty"`
}

// CreateGPUInstanceResponse 创建GPU实例响应
type CreateGPUInstanceResponse struct {
	ID string `json:"id"`
	Id string `json:"Id"`
}

// InstanceID 获取实例ID（兼容大小写）
func (r *CreateGPUInstanceResponse) InstanceID() string {
	if r.ID != "" {
		return r.ID
	}
	return r.Id
}

// CreateGPUInstance 创建GPU实例
func (c *Client) CreateGPUInstance(ctx context.Context, req CreateGPUInstanceRequest) (*CreateGPUInstanceResponse, error) {
	respBody, err := c.doRequest(ctx, "POST", "/gpu/instance/create", req)
	if err != nil {
		return nil, err
	}

	var result CreateGPUInstanceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// PortMapping 端口映射
type PortMapping struct {
	Port     int    `json:"port"`
	Endpoint string `json:"endpoint"`
	Type     string `json:"type"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
	ID string `json:"id"`
	IP string `json:"ip"`
}

// StatusError 状态错误
type StatusError struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

// GPUInstance GPU实例信息
type GPUInstance struct {
	ID            string        `json:"id"`
	Id            string        `json:"Id"`
	Name          string        `json:"name"`
	ClusterId     string        `json:"clusterId"`
	ClusterName   string        `json:"clusterName"`
	Status        string        `json:"status"`
	ImageUrl      string        `json:"imageUrl"`
	ProductId     string        `json:"productId"`
	ProductName   string        `json:"productName"`
	GpuNum        string        `json:"gpuNum"`
	RootfsSize    int           `json:"rootfsSize"`
	PortMappings  []PortMapping `json:"portMappings"`
	SshCommand    string        `json:"sshCommand"`
	Password      string        `json:"password"`
	Network       *NetworkInfo  `json:"network"`
	StatusError   *StatusError  `json:"statusError"`
	CreatedAt     string        `json:"createdAt"`
	LastStartedAt string        `json:"lastStartedAt"`
	BillingMode   string        `json:"billingMode"`
	EndTime       string        `json:"endTime"`
}

// GetID 获取实例ID（兼容大小写）
func (g *GPUInstance) GetID() string {
	if g.ID != "" {
		return g.ID
	}
	return g.Id
}

// ListGPUInstancesResponse 实例列表响应
type ListGPUInstancesResponse struct {
	Instances []GPUInstance `json:"instances"`
}

// ListGPUInstances 查询实例列表
func (c *Client) ListGPUInstances(ctx context.Context) ([]GPUInstance, error) {
	respBody, err := c.doRequest(ctx, "GET", "/gpu/instances", nil)
	if err != nil {
		return nil, err
	}

	var result ListGPUInstancesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Instances, nil
}

// GetGPUInstanceResponse 实例详情响应
type GetGPUInstanceResponse struct {
	Instance GPUInstance `json:"instance"`
}

// GetGPUInstance 查询实例详情
func (c *Client) GetGPUInstance(ctx context.Context, instanceID string) (*GPUInstance, error) {
	respBody, err := c.doRequest(ctx, "GET", "/gpu/instance?instanceId="+instanceID, nil)
	if err != nil {
		return nil, err
	}
	// PPIO 可能直接返回实例对象或包裹在 instance 字段中
	var direct GPUInstance
	if err := json.Unmarshal(respBody, &direct); err == nil && direct.GetID() != "" {
		return &direct, nil
	}

	var wrapped GetGPUInstanceResponse
	if err := json.Unmarshal(respBody, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &wrapped.Instance, nil
}

// StartGPUInstanceRequest 启动实例请求
type StartGPUInstanceRequest struct {
	InstanceID string `json:"instanceId"`
}

// StartGPUInstance 启动实例
func (c *Client) StartGPUInstance(ctx context.Context, instanceID string) error {
	_, err := c.doRequest(ctx, "POST", "/gpu/instance/start", StartGPUInstanceRequest{InstanceID: instanceID})
	return err
}

// StopGPUInstanceRequest 停止实例请求
type StopGPUInstanceRequest struct {
	InstanceID string `json:"instanceId"`
}

// StopGPUInstance 停止实例
func (c *Client) StopGPUInstance(ctx context.Context, instanceID string) error {
	_, err := c.doRequest(ctx, "POST", "/gpu/instance/stop", StopGPUInstanceRequest{InstanceID: instanceID})
	return err
}

// DeleteGPUInstanceRequest 删除实例请求
type DeleteGPUInstanceRequest struct {
	InstanceID string `json:"instanceId"`
}

// DeleteGPUInstance 删除实例
func (c *Client) DeleteGPUInstance(ctx context.Context, instanceID string) error {
	_, err := c.doRequest(ctx, "POST", "/gpu/instance/delete", DeleteGPUInstanceRequest{InstanceID: instanceID})
	return err
}
