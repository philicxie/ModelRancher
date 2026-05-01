package vastai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://console.vast.ai/api/v0"
)

// Client Vast.ai API客户端
type Client struct {
	apiKey     string
	sshKey     string
	sshKeyID   string
	httpClient *http.Client
}

// Config Vast.ai客户端配置
type Config struct {
	APIKey   string
	SSHKey   string // SSH私钥内容
	SSHKeyID string // SSH公钥ID
}

// NewClient 创建Vast.ai客户端
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &Client{
		apiKey:   cfg.APIKey,
		sshKey:   cfg.SSHKey,
		sshKeyID: cfg.SSHKeyID,
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

// OfferFilter 搜索过滤条件
type OfferFilter struct {
	GPUName     string  `json:"gpu_name,omitempty"`
	NumGPUs     int     `json:"num_gpus,omitempty"`
	MinGPUs     int     `json:"gpu_ram,omitempty"`
	Reliability float64 `json:"reliability,omitempty"`
	Verified    bool    `json:"verified,omitempty"`
	Rentable    bool    `json:"rentable,omitempty"`
	Type        string  `json:"type,omitempty"` // "ondemand" or "bid"
	Limit       int     `json:"limit,omitempty"`
}

// Offer 机器提供信息
type Offer struct {
	ID          int64   `json:"id"`
	GPUName     string  `json:"gpu_name"`
	NumGPUs     int     `json:"num_gpus"`
	GPURAM      int     `json:"gpu_ram"`
	Reliability float64 `json:"reliability"`
	DPHTotal    float64 `json:"dph_total"` // $/hour
	DiskSpace   float64 `json:"disk_space"`
	inetDown    int     `json:"inet_down"`
	inetUp      int     `json:"inet_up"`
	MachineID   int     `json:"machine_id"`
	Country     string  `json:"country"`
	Verified    bool    `json:"verified"`
}

// BundlesResponse 搜索结果响应
type BundlesResponse struct {
	Offers []Offer `json:"offers"`
	Total  int     `json:"total"`
}

// SearchOffers 搜索可用的GPU实例
func (c *Client) SearchOffers(ctx context.Context, filter OfferFilter) ([]Offer, error) {
	filterBody := map[string]interface{}{}

	if filter.GPUName != "" {
		filterBody["gpu_name"] = map[string][]string{"in": {filter.GPUName}}
	}
	if filter.NumGPUs > 0 {
		filterBody["num_gpus"] = filter.NumGPUs
	}
	if filter.MinGPUs > 0 {
		filterBody["gpu_ram"] = map[string]int{"gte": filter.MinGPUs}
	}
	if filter.Reliability > 0 {
		filterBody["reliability"] = map[string]float64{"gte": filter.Reliability}
	}
	if filter.Type != "" {
		filterBody["type"] = filter.Type
	}
	filterBody["verified"] = map[string]bool{"eq": filter.Verified}
	filterBody["rentable"] = map[string]bool{"eq": filter.Rentable}
	if filter.Limit > 0 {
		filterBody["limit"] = filter.Limit
	} else {
		filterBody["limit"] = 10
	}

	respBody, err := c.doRequest(ctx, "POST", "/bundles/", filterBody)
	if err != nil {
		return nil, err
	}

	var result BundlesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Offers, nil
}

// CreateInstanceRequest 创建实例请求
type CreateInstanceRequest struct {
	Image   string            `json:"image"`              // Docker镜像
	Label   string            `json:"label,omitempty"`    // 实例标签
	Disk    int               `json:"disk,omitempty"`     // 磁盘大小(GB)
	Runtype string            `json:"runtype"`            // ssh_direct, jupyter_direct, args
	Env     map[string]string `json:"env,omitempty"`      // 环境变量
	Onstart string            `json:"onstart,omitempty"`  // 启动命令
	ArgsStr string            `json:"args_str,omitempty"` // 替换Docker CMD
}

// CreateInstanceResponse 创建实例响应
type CreateInstanceResponse struct {
	NewContract int64 `json:"new_contract"`
	InstanceID  int64 `json:"instance_id"`
}

// CreateInstance 创建GPU实例
func (c *Client) CreateInstance(ctx context.Context, offerID int64, req CreateInstanceRequest) (*CreateInstanceResponse, error) {
	if req.Runtype == "" {
		req.Runtype = "ssh_direct"
	}

	body := map[string]interface{}{
		"image":   req.Image,
		"runtype": req.Runtype,
	}

	if req.Label != "" {
		body["label"] = req.Label
	}
	if req.Disk > 0 {
		body["disk"] = req.Disk
	}
	if len(req.Env) > 0 {
		body["env"] = req.Env
	}
	if req.Onstart != "" {
		body["onstart"] = req.Onstart
	}
	if req.ArgsStr != "" {
		body["args_str"] = req.ArgsStr
	}

	respBody, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/asks/%d/", offerID), body)
	if err != nil {
		return nil, err
	}

	var result CreateInstanceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// Instance 实例信息
type Instance struct {
	ID             int64   `json:"id"`
	Label          string  `json:"label"`
	Status         string  `json:"status"`
	ActualStatus   string  `json:"actual_status"`
	SSHHost        string  `json:"ssh_host"`
	SSHPort        int     `json:"ssh_port"`
	InternalIP     string  `json:"internal_ip"`
	PublicHostaddr string  `json:"public_hostaddr"`
	MachineID      int     `json:"machine_id"`
	Image          string  `json:"image"`
	// Metrics fields (current snapshot from Vast.ai)
	CPUUtil      float64 `json:"cpu_util"`
	MemUsage     float64 `json:"mem_usage"`
	MemLimit     float64 `json:"mem_limit"`
	VMemUsage    float64 `json:"vmem_usage"`
	GPUUtil      float64 `json:"gpu_util"`
	GPUTemp      float64 `json:"gpu_temp"`
	GPURam       int     `json:"gpu_ram"`
	GPUTotalRam  int     `json:"gpu_totalram"`
	DiskUtil     float64 `json:"disk_util"`
	DiskUsage    float64 `json:"disk_usage"`
	DiskSpace    float64 `json:"disk_space"`
	DiskBW       float64 `json:"disk_bw"`
}

// ShowInstancesResponse 显示实例响应
type ShowInstancesResponse struct {
	Instances []Instance `json:"instances"`
}

// ShowInstances 显示当前用户的所有实例
func (c *Client) ShowInstances(ctx context.Context) ([]Instance, error) {
	respBody, err := c.doRequest(ctx, "GET", "/instances/", nil)
	if err != nil {
		return nil, err
	}

	var result ShowInstancesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Instances, nil
}

// GetInstance 获取指定实例信息
func (c *Client) GetInstance(ctx context.Context, instanceID int64) (*Instance, error) {
	respBody, err := c.doRequest(ctx, "GET", fmt.Sprintf("/instances/%d/", instanceID), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Instances Instance `json:"instances"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result.Instances, nil
}

// DestroyInstance 销毁实例
func (c *Client) DestroyInstance(ctx context.Context, instanceID int64) error {
	_, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/instances/%d/", instanceID), nil)
	return err
}

// StartInstance 启动实例
func (c *Client) StartInstance(ctx context.Context, instanceID int64) error {
	body := map[string]string{"target_state": "running"}
	_, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/instances/%d/", instanceID), body)
	return err
}

// StopInstance 停止实例
func (c *Client) StopInstance(ctx context.Context, instanceID int64) error {
	body := map[string]string{"target_state": "stopped"}
	_, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/instances/%d/", instanceID), body)
	return err
}

// GetInstanceMetrics 获取实例当前指标快照（Vast.ai 没有时序 API，返回单点数据）
func (c *Client) GetInstanceMetrics(ctx context.Context, instanceID int64) (*Instance, error) {
	return c.GetInstance(ctx, instanceID)
}

// RebootInstance 重启实例
func (c *Client) RebootInstance(ctx context.Context, instanceID int64) error {
	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/instances/%d/rescue/", instanceID), nil)
	return err
}
