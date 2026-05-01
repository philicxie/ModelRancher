package service

import (
	"context"
	"fmt"

	"ml-platform/internal/executor/provider"
)

// GPUService GPU实例服务
type GPUService struct {
	providerManager *provider.ProviderManager
}

// NewGPUService 创建GPU服务
func NewGPUService(pm *provider.ProviderManager) *GPUService {
	return &GPUService{providerManager: pm}
}

// GPUInstance GPU实例信息
type GPUInstance struct {
	ID          string  `json:"id"`
	FlavorID    string  `json:"flavor_id"`
	Provider    string  `json:"provider"`
	GPUType     string  `json:"gpu_type"`
	NumGPUs     int     `json:"num_gpus"`
	GPURAM      int     `json:"gpu_ram"`
	DiskSpace   float64 `json:"disk_space"`
	Reliability float64 `json:"reliability"`
	Price       float64 `json:"price"`
	Location    string  `json:"location"`
}

// SearchGPUInstancesRequest 搜索GPU实例请求
type SearchGPUInstancesRequest struct {
	GPUType  string  `json:"gpu_type" form:"gpu_type"`
	MinGPUs  int     `json:"min_gpus" form:"min_gpus"`
	MinRAM   int     `json:"min_ram" form:"min_ram"`
	MaxPrice float64 `json:"max_price" form:"max_price"`
	Provider string  `json:"provider" form:"provider"`
}

// SearchGPUInstances 搜索可用的GPU实例列表
func (s *GPUService) SearchGPUInstances(ctx context.Context, req *SearchGPUInstancesRequest) ([]GPUInstance, error) {
	if s.providerManager == nil {
		return nil, fmt.Errorf("provider manager not initialized")
	}

	resourceReq := &provider.ResourceRequest{
		GPUType:  req.GPUType,
		MinRAM:   req.MinRAM,
		MaxPrice: req.MaxPrice,
		Provider: req.Provider,
	}

	resources := s.providerManager.ListAllResources(ctx, resourceReq)

	instances := make([]GPUInstance, 0, len(resources))
	for _, r := range resources {
		instances = append(instances, GPUInstance{
			ID:          r.ID,
			FlavorID:    fmt.Sprintf("%s-%s-%d", r.Provider, r.GPUType, r.NumGPUs),
			Provider:    r.Provider,
			GPUType:     r.GPUType,
			NumGPUs:     r.NumGPUs,
			GPURAM:      r.GPURAM,
			DiskSpace:   r.DiskSpace,
			Reliability: r.Reliability,
			Price:       r.Price,
			Location:    r.Location,
		})
	}

	return instances, nil
}

// GetProviderStats 获取所有Provider的统计信息
func (s *GPUService) GetProviderStats(ctx context.Context) map[string]*provider.Stats {
	if s.providerManager == nil {
		return nil
	}
	return s.providerManager.GetAllStats(ctx)
}

// ProviderInfo 可用Provider信息
type ProviderInfo struct {
	Name     string `json:"name"`
	Available bool  `json:"available"`
}

// ListProviders 列出所有可用的Provider
func (s *GPUService) ListProviders(ctx context.Context) []ProviderInfo {
	if s.providerManager == nil {
		return nil
	}

	available := s.providerManager.ListAvailableProviders()
	infos := make([]ProviderInfo, 0, len(available))
	for _, name := range available {
		infos = append(infos, ProviderInfo{
			Name:      name,
			Available: true,
		})
	}
	return infos
}
