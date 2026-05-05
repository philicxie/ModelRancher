package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ml-platform/internal/executor/provider/ppio"
	"ml-platform/internal/model"
	"ml-platform/internal/repository"
)

// DockerHubSearchResult Docker Hub 搜索结果
type DockerHubSearchResult struct {
	Count   int `json:"count"`
	Page    int `json:"page"`
	Results []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StarCount   int    `json:"star_count"`
		IsOfficial  bool   `json:"is_official"`
		IsTrusted   bool   `json:"is_trusted"`
	} `json:"results"`
}

// PublicImage 公开镜像信息
type PublicImage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StarCount   int    `json:"star_count"`
	IsOfficial  bool   `json:"is_official"`
	IsFavorited bool   `json:"is_favorited"`
}

// PrivateImage 私有镜像信息
type PrivateImage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
	Visibility  string `json:"visibility"`
}

// 内置热门训练镜像列表（本地搜索源，不依赖外部网络）
var builtInImages = []struct {
	Name        string
	Description string
	StarCount   int
	IsOfficial  bool
}{
	// PyTorch 系列
	{"pytorch/pytorch", "PyTorch 官方镜像，包含最新稳定版", 28000, true},
	{"pytorch/pytorch:2.2.0-cuda12.1-cudnn8-runtime", "PyTorch 2.2 + CUDA 12.1 运行时", 28000, true},
	{"pytorch/pytorch:2.1.0-cuda11.8-cudnn8-runtime", "PyTorch 2.1 + CUDA 11.8 运行时", 28000, true},
	{"pytorch/pytorch:2.0.1-cuda11.7-cudnn8-runtime", "PyTorch 2.0 + CUDA 11.7 运行时", 28000, true},
	{"pytorch/pytorch:1.13.1-cuda11.6-cudnn8-runtime", "PyTorch 1.13 + CUDA 11.6 运行时", 28000, true},
	{"pytorch/pytorch:2.2.0-cuda12.1-cudnn8-devel", "PyTorch 2.2 + CUDA 12.1 开发版", 28000, true},
	
	// TensorFlow 系列
	{"tensorflow/tensorflow", "TensorFlow 官方镜像", 22000, true},
	{"tensorflow/tensorflow:latest-gpu", "TensorFlow 最新 GPU 版本", 22000, true},
	{"tensorflow/tensorflow:2.15.0-gpu", "TensorFlow 2.15 GPU 版本", 22000, true},
	{"tensorflow/tensorflow:2.14.0-gpu", "TensorFlow 2.14 GPU 版本", 22000, true},
	{"tensorflow/tensorflow:2.13.0-gpu", "TensorFlow 2.13 GPU 版本", 22000, true},
	
	// NVIDIA CUDA 基础镜像
	{"nvidia/cuda", "NVIDIA CUDA 官方基础镜像", 18000, true},
	{"nvidia/cuda:12.1.0-cudnn8-devel-ubuntu22.04", "CUDA 12.1 + cuDNN 8 + Ubuntu 22.04 开发版", 18000, true},
	{"nvidia/cuda:11.8.0-cudnn8-devel-ubuntu22.04", "CUDA 11.8 + cuDNN 8 + Ubuntu 22.04 开发版", 18000, true},
	{"nvidia/cuda:12.1.0-base-ubuntu22.04", "CUDA 12.1 基础版 + Ubuntu 22.04", 18000, true},
	
	// 深度学习框架
	{"huggingface/transformers-pytorch-gpu", "HuggingFace Transformers + PyTorch GPU", 8500, false},
	{"huggingface/transformers-tensorflow-gpu", "HuggingFace Transformers + TensorFlow GPU", 6200, false},
	{"deepspeed/deepspeed", "Microsoft DeepSpeed 分布式训练", 4500, false},
	{"mosaicml/pytorch", "MosaicML PyTorch 训练镜像", 3200, false},
	
	// JAX / XLA
	{"google/jax", "Google JAX 官方镜像", 5500, true},
	{"google/jax:latest-cuda", "JAX + CUDA 最新版", 5500, true},
	
	// MXNet
	{"apache/mxnet", "Apache MXNet 官方镜像", 4000, true},
	{"apache/mxnet:1.9.1-cu112", "MXNet 1.9.1 CUDA 11.2", 4000, true},
	
	// PaddlePaddle
	{"paddlepaddle/paddle", "百度 PaddlePaddle 官方镜像", 3800, false},
	{"paddlepaddle/paddle:2.6.0-gpu-cuda11.7-cudnn8.4-trt8.4", "PaddlePaddle 2.6 GPU 版", 3800, false},
	{"paddlepaddle/paddle:2.5.2-gpu-cuda11.7-cudnn8.4-trt8.4", "PaddlePaddle 2.5 GPU 版", 3800, false},
	
	// OneFlow
	{"oneflowinc/oneflow", "OneFlow 深度学习框架", 2100, false},
	
	// MindSpore
	{"mindspore/mindspore-gpu", "华为 MindSpore GPU 版", 1800, false},
	
	// 计算机视觉
	{"ultralytics/ultralytics", "YOLOv8 Ultralytics 官方镜像", 7200, false},
	{"openmmlab/pytorch", "OpenMMLab PyTorch 基础镜像", 5600, false},
	{"mmclassification/pytorch", "OpenMMLab 图像分类", 3400, false},
	{"mmdetection/pytorch", "OpenMMLab 目标检测", 4100, false},
	{"mmsegmentation/pytorch", "OpenMMLab 语义分割", 2800, false},
	
	// NLP / LLM
	{"huggingface/accelerate-gpu", "HuggingFace Accelerate GPU", 4800, false},
	{"vllm/vllm-openai", "vLLM 高性能推理服务", 3900, false},
	{"textgenwebui/text-generation-webui", "Text Generation WebUI", 3500, false},
	
	// 分布式训练
	{"horovod/horovod", "Horovod 分布式训练框架", 2900, false},
	{"pytorch/torchdistx", "PyTorch 分布式扩展", 1500, false},
	
	// Jupyter / 开发环境
	{"jupyter/pytorch-notebook", "Jupyter PyTorch Notebook", 12000, true},
	{"jupyter/tensorflow-notebook", "Jupyter TensorFlow Notebook", 11000, true},
	{"jupyter/datascience-notebook", "Jupyter 数据科学环境", 9500, true},
	{"jupyter/scipy-notebook", "Jupyter SciPy 环境", 8000, true},
	
	// Python 基础
	{"python", "Python 官方镜像", 45000, true},
	{"python:3.11-slim", "Python 3.11 精简版", 45000, true},
	{"python:3.10-slim", "Python 3.10 精简版", 45000, true},
	{"python:3.9-slim", "Python 3.9 精简版", 45000, true},
	{"python:3.11", "Python 3.11 完整版", 45000, true},
	
	// Ubuntu 基础
	{"ubuntu", "Ubuntu 官方镜像", 60000, true},
	{"ubuntu:22.04", "Ubuntu 22.04 LTS", 60000, true},
	{"ubuntu:20.04", "Ubuntu 20.04 LTS", 60000, true},
	
	// Miniconda / Anaconda
	{"continuumio/miniconda3", "Miniconda3 官方镜像", 15000, true},
	{"continuumio/anaconda3", "Anaconda3 官方镜像", 12000, true},
	
	// 其他常用 ML 工具
	{"rapidsai/rapidsai-core", "RAPIDS AI GPU 数据科学", 4200, false},
	{"kubeflow/kubeflow", "Kubeflow 机器学习平台", 7800, false},
	{"mlflow/mlflow", "MLflow 实验追踪", 6500, false},
	{"databricksruntime/gpu-tensorflow", "Databricks GPU TensorFlow", 2100, false},
	{"databricksruntime/gpu-pytorch", "Databricks GPU PyTorch", 1800, false},
	
	// 推理部署
	{"nvcr.io/nvidia/tritonserver", "NVIDIA Triton 推理服务器", 3500, false},
	{"nvcr.io/nvidia/tensorrt", "NVIDIA TensorRT", 2800, false},
	{"onnxruntime/gpu", "ONNX Runtime GPU", 2200, false},
	
	// 向量数据库 / RAG
	{"chromadb/chroma", "ChromaDB 向量数据库", 3100, false},
	{"qdrant/qdrant", "Qdrant 向量搜索引擎", 2900, false},
	{"milvusdb/milvus", "Milvus 向量数据库", 2600, false},
	
	// 常用工具
	{"nginx", "Nginx 官方镜像", 55000, true},
	{"redis", "Redis 官方镜像", 42000, true},
	{"postgres", "PostgreSQL 官方镜像", 38000, true},
	{"mysql", "MySQL 官方镜像", 35000, true},
	{"mongo", "MongoDB 官方镜像", 28000, true},
	{"node", "Node.js 官方镜像", 32000, true},
	{"golang", "Go 官方镜像", 24000, true},
	{"rust", "Rust 官方镜像", 18000, true},
}

// ImageService 镜像服务
type ImageService struct {
	favoriteRepo   *repository.ImageFavoriteRepository
	httpClient     *http.Client
	githubToken    string
	githubUsername string
}

// NewImageService 创建镜像服务
func NewImageService(favoriteRepo *repository.ImageFavoriteRepository) *ImageService {
	return &ImageService{
		favoriteRepo:   favoriteRepo,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		githubToken:    os.Getenv("GITHUB_TOKEN"),
		githubUsername: os.Getenv("GITHUB_USERNAME"),
	}
}

// SearchPublicImages 搜索公开镜像
// 优先从内置列表过滤（不依赖网络），同时并行尝试 Docker Hub API 获取更多结果
func (s *ImageService) SearchPublicImages(ctx context.Context, query string, page, pageSize int) (*DockerHubSearchResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 1. 先从内置列表过滤（本地搜索，即时返回）
	localResults := s.searchBuiltIn(query)

	// 2. 并行尝试 Docker Hub API，最多等 2 秒
	hubCh := make(chan []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StarCount   int    `json:"star_count"`
		IsOfficial  bool   `json:"is_official"`
		IsTrusted   bool   `json:"is_trusted"`
	}, 1)
	go func() {
		hubCh <- s.searchDockerHub(ctx, query, page, pageSize)
	}()

	var hubResults []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StarCount   int    `json:"star_count"`
		IsOfficial  bool   `json:"is_official"`
		IsTrusted   bool   `json:"is_trusted"`
	}
	select {
	case hubResults = <-hubCh:
	case <-time.After(2 * time.Second):
		// Docker Hub 超时，只用本地结果
	}

	// 3. 合并结果，去重（内置列表优先）
	seen := make(map[string]bool)
	merged := make([]struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StarCount   int    `json:"star_count"`
		IsOfficial  bool   `json:"is_official"`
		IsTrusted   bool   `json:"is_trusted"`
	}, 0, len(localResults)+len(hubResults))

	for _, img := range localResults {
		if seen[img.Name] {
			continue
		}
		seen[img.Name] = true
		merged = append(merged, struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			StarCount   int    `json:"star_count"`
			IsOfficial  bool   `json:"is_official"`
			IsTrusted   bool   `json:"is_trusted"`
		}{
			Name:        img.Name,
			Description: img.Description,
			StarCount:   img.StarCount,
			IsOfficial:  img.IsOfficial,
			IsTrusted:   img.IsOfficial,
		})
	}

	for _, img := range hubResults {
		if seen[img.Name] {
			continue
		}
		seen[img.Name] = true
		merged = append(merged, img)
	}

	// 分页
	total := len(merged)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &DockerHubSearchResult{
		Count:   total,
		Page:    page,
		Results: merged[start:end],
	}, nil
}

// searchBuiltIn 从内置列表中搜索镜像（不区分大小写，匹配名称和描述）
func (s *ImageService) searchBuiltIn(query string) []struct {
	Name        string
	Description string
	StarCount   int
	IsOfficial  bool
} {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return builtInImages
	}

	results := make([]struct {
		Name        string
		Description string
		StarCount   int
		IsOfficial  bool
	}, 0)

	for _, img := range builtInImages {
		nameLower := strings.ToLower(img.Name)
		descLower := strings.ToLower(img.Description)
		if strings.Contains(nameLower, q) || strings.Contains(descLower, q) {
			results = append(results, img)
		}
	}

	return results
}

// searchDockerHub 调用 Docker Hub API 搜索（带独立超时，失败返回空）
func (s *ImageService) searchDockerHub(ctx context.Context, query string, page, pageSize int) []struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	StarCount   int    `json:"star_count"`
	IsOfficial  bool   `json:"is_official"`
	IsTrusted   bool   `json:"is_trusted"`
} {
	// 使用独立的短超时上下文，避免阻塞本地搜索结果
	shortCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://hub.docker.com/v2/search/repositories/?query=%s&page=%d&page_size=%d",
		query, page, pageSize)

	req, err := http.NewRequestWithContext(shortCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result DockerHubSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	return result.Results
}

// ListPrivateImages 查询私有镜像
// provider 为空时默认走 GitHub Container Registry；provider="ppio" 时走 PPIO API
func (s *ImageService) ListPrivateImages(ctx context.Context, provider string) ([]*PrivateImage, error) {
	if provider == "ppio" {
		return s.listPPIOImages(ctx)
	}
	return s.listGitHubImages(ctx)
}

// listPPIOImages 通过 PPIO API 获取私有镜像
func (s *ImageService) listPPIOImages(ctx context.Context) ([]*PrivateImage, error) {
	apiKey := os.Getenv("PPIO_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("PPIO_API_KEY not configured")
	}

	ppioClient, err := ppio.NewClient(ppio.Config{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("failed to create ppio client: %w", err)
	}

	images, err := ppioClient.ListImages(ctx, "private", "", 100, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to list ppio images: %w", err)
	}

	var result []*PrivateImage
	for _, img := range images {
		// PPIO 镜像按 tag 展开，每个 tag 作为一条记录
		if len(img.Tags) > 0 {
			for _, tag := range img.Tags {
				result = append(result, &PrivateImage{
					Name:        tag.URL,
					Description: fmt.Sprintf("PPIO 私有镜像: %s", img.Name),
					UpdatedAt:   "",
					Visibility:  "private",
				})
			}
		} else {
			result = append(result, &PrivateImage{
				Name:        img.URL,
				Description: fmt.Sprintf("PPIO 私有镜像: %s", img.Name),
				UpdatedAt:   "",
				Visibility:  "private",
			})
		}
	}
	return result, nil
}

// listGitHubImages 通过 GitHub REST API 查询账户下的 container packages
// 若配置了 GITHUB_TOKEN，可获取包括私有在内的所有 packages；否则只获取公开 packages
func (s *ImageService) listGitHubImages(ctx context.Context) ([]*PrivateImage, error) {
	// 确定 API 路径：有 token 走 /user/packages（认证用户），否则走 /users/{username}/packages
	var url string
	if s.githubToken != "" {
		url = "https://api.github.com/user/packages?package_type=container"
	} else {
		username := s.githubUsername
		if username == "" {
			username = "philicxie"
		}
		url = fmt.Sprintf("https://api.github.com/users/%s/packages?package_type=container", username)
	}

	packages, err := s.fetchGitHubPackages(ctx, url)
	if err != nil {
		return s.mockPrivateImages(), nil
	}
	if len(packages) == 0 {
		return s.mockPrivateImages(), nil
	}

	// 为每个 package 获取版本列表，构建完整镜像名
	result := make([]*PrivateImage, 0, len(packages))
	for _, pkg := range packages {
		tags := s.fetchPackageVersions(ctx, pkg.Owner, pkg.Name)
		imageName := fmt.Sprintf("ghci.io/%s/%s", pkg.Owner, pkg.Name)

		if len(tags) > 0 {
			// 展示带 tag 的完整镜像名
			for _, tag := range tags {
				result = append(result, &PrivateImage{
					Name:        fmt.Sprintf("%s:%s", imageName, tag),
					Description: fmt.Sprintf("GitHub Container Registry - %s", pkg.Name),
					UpdatedAt:   pkg.UpdatedAt,
					Visibility:  pkg.Visibility,
				})
			}
		} else {
			// 没有版本信息时展示 latest
			result = append(result, &PrivateImage{
				Name:        imageName + ":latest",
				Description: fmt.Sprintf("GitHub Container Registry - %s", pkg.Name),
				UpdatedAt:   pkg.UpdatedAt,
				Visibility:  pkg.Visibility,
			})
		}
	}

	return result, nil
}

// githubPackage GitHub API 返回的 package 基本信息
type githubPackage struct {
	Name       string `json:"name"`
	Owner      string `json:"-"` // 从 owner.login 解析
	UpdatedAt  string `json:"updated_at"`
	Visibility string `json:"visibility"`
	HtmlURL    string `json:"html_url"`
}

// fetchGitHubPackages 调用 GitHub API 获取 packages 列表
func (s *ImageService) fetchGitHubPackages(ctx context.Context, url string) ([]githubPackage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if s.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.githubToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var raw []struct {
		Name       string `json:"name"`
		UpdatedAt  string `json:"updated_at"`
		Visibility string `json:"visibility"`
		HtmlURL    string `json:"html_url"`
		Owner      struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	packages := make([]githubPackage, 0, len(raw))
	for _, r := range raw {
		owner := r.Owner.Login
		if owner == "" {
			// 对于 /user/packages 接口，owner 可能在不同字段
			// 尝试从 html_url 解析: https://github.com/OWNER/packages/container/NAME
			parts := strings.Split(r.HtmlURL, "/")
			if len(parts) >= 4 {
				owner = parts[3]
			}
		}
		if owner == "" {
			owner = s.githubUsername
		}
		if owner == "" {
			owner = "philicxie"
		}
		packages = append(packages, githubPackage{
			Name:       r.Name,
			Owner:      owner,
			UpdatedAt:  r.UpdatedAt,
			Visibility: r.Visibility,
			HtmlURL:    r.HtmlURL,
		})
	}

	return packages, nil
}

// fetchPackageVersions 获取某个 package 的版本 tags
func (s *ImageService) fetchPackageVersions(ctx context.Context, owner, packageName string) []string {
	// 使用 /users/{username}/packages/{package_type}/{package_name}/versions 接口
	url := fmt.Sprintf("https://api.github.com/users/%s/packages/container/%s/versions", owner, packageName)
	if s.githubToken != "" {
		// 有 token 时尝试 /user/packages 路径（可能包含私有 package）
		url = fmt.Sprintf("https://api.github.com/user/packages/container/%s/versions", packageName)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if s.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.githubToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var versions []struct {
		Metadata struct {
			Container struct {
				Tags []string `json:"tags"`
			} `json:"container"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil
	}

	// 收集所有 tag，去重，最多取前 5 个
	seen := make(map[string]bool)
	tags := make([]string, 0)
	for _, v := range versions {
		for _, t := range v.Metadata.Container.Tags {
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			tags = append(tags, t)
			if len(tags) >= 5 {
				return tags
			}
		}
	}
	return tags
}

// mockPrivateImages 预设的私有镜像列表
func (s *ImageService) mockPrivateImages() []*PrivateImage {
	return []*PrivateImage{
		{
			Name:        "ghci.io/philicxie/pytorch-cuda:latest",
			Description: "PyTorch 2.1 + CUDA 12.1 训练环境",
			UpdatedAt:   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			Visibility:  "public",
		},
		{
			Name:        "ghci.io/philicxie/tensorflow-gpu:latest",
			Description: "TensorFlow 2.15 + CUDA 12.2 GPU 版本",
			UpdatedAt:   time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			Visibility:  "public",
		},
		{
			Name:        "ghci.io/philicxie/ml-base:ubuntu22.04",
			Description: "Ubuntu 22.04 基础 ML 环境，预装常用工具",
			UpdatedAt:   time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
			Visibility:  "public",
		},
		{
			Name:        "ghci.io/philicxie/deepspeed:v0.12",
			Description: "DeepSpeed 分布式训练环境",
			UpdatedAt:   time.Now().Add(-96 * time.Hour).Format(time.RFC3339),
			Visibility:  "public",
		},
		{
			Name:        "ghci.io/philicxie/jax-cuda:latest",
			Description: "JAX + CUDA 高性能计算环境",
			UpdatedAt:   time.Now().Add(-120 * time.Hour).Format(time.RFC3339),
			Visibility:  "public",
		},
	}
}

// AddFavorite 添加收藏
func (s *ImageService) AddFavorite(ctx context.Context, userID, imageName, description string, starCount int, isOfficial bool) (*model.ImageFavorite, error) {
	// 检查是否已收藏
	existing, _ := s.favoriteRepo.GetByUserAndImage(ctx, userID, imageName)
	if existing != nil {
		return nil, fmt.Errorf("image already favorited")
	}

	fav := model.NewImageFavorite(userID, imageName, description, starCount, isOfficial)
	if err := s.favoriteRepo.Create(ctx, fav); err != nil {
		return nil, err
	}
	return fav, nil
}

// RemoveFavorite 取消收藏
func (s *ImageService) RemoveFavorite(ctx context.Context, userID, imageName string) error {
	return s.favoriteRepo.DeleteByUserAndImage(ctx, userID, imageName)
}

// ListFavorites 列出用户收藏的镜像
func (s *ImageService) ListFavorites(ctx context.Context, userID string) ([]*model.ImageFavorite, error) {
	return s.favoriteRepo.ListByUser(ctx, userID)
}

// CheckFavorited 检查镜像是否已收藏
func (s *ImageService) CheckFavorited(ctx context.Context, userID, imageName string) bool {
	fav, _ := s.favoriteRepo.GetByUserAndImage(ctx, userID, imageName)
	return fav != nil
}
