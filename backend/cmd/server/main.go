package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"ml-platform/internal/api"
	"ml-platform/internal/executor"
	"ml-platform/internal/executor/provider"
	"ml-platform/internal/executor/provider/dummy"
	"ml-platform/internal/executor/provider/ppio"
	"ml-platform/internal/executor/provider/vastai"
	"ml-platform/internal/model"
	"ml-platform/internal/repository"
	"ml-platform/internal/service"
	"ml-platform/pkg/cosclient"
	"ml-platform/pkg/database"

	"gorm.io/gorm"
)

func main() {
	// 配置日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println(strings.Repeat("=", 60))
	log.Println("ML Training Platform - Starting...")
	log.Println(strings.Repeat("=", 60))

	// 0. 加载 .env 文件（如果存在）
	if err := loadEnvFile(".env"); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}

	// 1. 数据库连接
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	// 2. 初始化任务仓储
	taskRepo := repository.NewTaskRepository(db)

	// 3. 腾讯云COS配置和自检
	cosClient, cosHealth := initCOS()
	if !cosHealth.OK {
		log.Printf("Warning: %s", cosHealth.Message)
		log.Printf("Storage operations may fail until Tencent Cloud is properly configured")
	}

	// 4. 初始化执行器
	execEngine := executor.NewExecutor(getEnv("DATA_DIR", "/app/data"), getEnv("OUTPUT_DIR", "/app/output"))

	// 5. 初始化服务层（使用数据库）
	taskService := service.NewTaskService(taskRepo, execEngine, cosClient)
	storageService := service.NewStorageService(cosClient)
	executorService := service.NewExecutorService(execEngine)

	// 6. 初始化GPU Provider管理器（统一由第一个成功初始化的provider创建）
	providerManager := provider.NewProviderManager()

	// 注册各 Provider（按配置条件）
	vastaiHealth := initVastai(providerManager)
	ppioHealth := initPPIO(providerManager)

	// 注册本地 Dummy Provider（总是注册，让用户作业可以跑在本地）
	dummyProvider := dummy.NewProvider(execEngine)
	providerManager.RegisterProvider("local", dummyProvider)
	log.Println("Local Dummy Provider registered")

	// 7. 初始化GPU服务
	gpuService := service.NewGPUService(providerManager)

	// 8. 初始化实例租赁服务
	instanceRepo := repository.NewInstanceRepository(db)
	instanceService := service.NewInstanceService(instanceRepo, providerManager)

	// 9. 运行启动自检（异步按项执行，15秒超时）
	runStartupChecks(taskService, cosClient, cosHealth, providerManager, vastaiHealth, ppioHealth)

	// 10. 初始化路由
	router := api.NewRouter(taskService, storageService, executorService, gpuService, instanceService)
	server := api.NewServer(router)

	// 11. 启动服务器
	port := getEnv("PORT", "8080")
	log.Printf("Server starting on port %s...", port)

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Received shutdown signal, closing gracefully...")
		server.Close()
	}()

	if err := server.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}

// initDatabase 初始化数据库连接
func initDatabase() (*gorm.DB, error) {
	log.Println("Initializing database connection...")

	cfg := database.DefaultConfig()
	cfg.Host = getEnv("DB_HOST", "localhost")
	cfg.Port = getEnv("DB_PORT", "5432")
	cfg.User = getEnv("DB_USER", "postgres")
	cfg.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.DBName = getEnv("DB_NAME", "ml_platform")

	db, err := database.Connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&model.Task{}, &model.TaskLog{}, &model.Instance{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	log.Printf("Database connected: %s:%s/%s", cfg.Host, cfg.Port, cfg.DBName)
	return db, nil
}

// COSHealth COS健康检查结果
type COSHealth struct {
	OK      bool
	Message string
}

// initCOS 初始化腾讯云COS客户端并返回健康状态
func initCOS() (*cosclient.Client, COSHealth) {
	secretID := getEnv("TENCENT_CLOUD_SECRET_ID", "")
	secretKey := getEnv("TENCENT_CLOUD_SECRET_KEY", "")
	bucket := getEnv("TENCENT_CLOUD_COS_BUCKET", "")
	region := getEnv("TENCENT_CLOUD_COS_REGION", "")

	log.Println("Checking Tencent Cloud COS configuration...")

	// 检查配置完整性
	missingFields := []string{}
	if secretID == "" {
		missingFields = append(missingFields, "TENCENT_CLOUD_SECRET_ID")
	}
	if secretKey == "" {
		missingFields = append(missingFields, "TENCENT_CLOUD_SECRET_KEY")
	}
	if bucket == "" {
		missingFields = append(missingFields, "TENCENT_CLOUD_COS_BUCKET")
	}
	if region == "" {
		missingFields = append(missingFields, "TENCENT_CLOUD_COS_REGION")
	}

	if len(missingFields) > 0 {
		return nil, COSHealth{
			OK:      false,
			Message: fmt.Sprintf("COS configuration incomplete. Missing: %v", missingFields),
		}
	}

	// 创建客户端
	client, err := cosclient.NewCOSClient(secretID, secretKey, bucket, region)
	if err != nil {
		return nil, COSHealth{
			OK:      false,
			Message: fmt.Sprintf("COS client initialization failed: %v", err),
		}
	}

	log.Printf("COS configured: bucket=%s, region=%s", bucket, region)

	// 测试连接 - 列出bucket内容
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objects, err := client.ListObjects(ctx, "")
	if err != nil {
		return client, COSHealth{
			OK:      false,
			Message: fmt.Sprintf("COS connection test failed: %v", err),
		}
	}

	log.Printf("COS connection verified: found %d objects in bucket", len(objects))

	return client, COSHealth{
		OK:      true,
		Message: fmt.Sprintf("COS connection successful. Bucket contains %d objects", len(objects)),
	}
}

// runStartupChecks 运行启动自检（异步按项执行，15秒超时）
func runStartupChecks(taskService *service.TaskService, cosClient *cosclient.Client, cosHealth COSHealth, providerManager *provider.ProviderManager, vastaiHealth *VastaiHealth, ppioHealth *PPIOHealth) {
	log.Println(strings.Repeat("=", 60))
	log.Println("Running startup checks...")
	log.Println(strings.Repeat("=", 60))

	allPassed := true

	// 1. 数据库连接检查
	log.Printf("[CHECK] Database: OK (connection established)")

	// 2. 腾讯云COS检查
	if cosHealth.OK {
		log.Printf("[CHECK] Tencent Cloud COS: OK (%s)", cosHealth.Message)
	} else {
		log.Printf("[CHECK] Tencent Cloud COS: FAILED - %s", cosHealth.Message)
		allPassed = false
	}

	// 3. Docker检查
	dockerOK := checkDocker()
	if dockerOK {
		log.Printf("[CHECK] Docker: OK (Docker daemon is running)")
	} else {
		log.Printf("[CHECK] Docker: WARNING (Docker daemon may not be running)")
	}

	// 4. 目录检查
	dataDir := getEnv("DATA_DIR", "/app/data")
	outputDir := getEnv("OUTPUT_DIR", "/app/output")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("[CHECK] Data Directory %s: WARNING (failed to create: %v)", dataDir, err)
	} else {
		log.Printf("[CHECK] Data Directory %s: OK", dataDir)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("[CHECK] Output Directory %s: WARNING (failed to create: %v)", outputDir, err)
	} else {
		log.Printf("[CHECK] Output Directory %s: OK", outputDir)
	}

	// 5. GPU Provider 异步健康检查（15秒超时）
	if providerManager != nil {
		registered := providerManager.ListProviders()
		log.Printf("[CHECK] Registered GPU providers: %v", registered)

		var wg sync.WaitGroup
		checkResults := make(map[string]bool)
		var mu sync.Mutex

		for _, name := range registered {
			p, ok := providerManager.GetProvider(name)
			if !ok {
				continue
			}

			wg.Add(1)
			go func(providerName string, prov provider.ExecutorProvider) {
				defer wg.Done()

				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				err := prov.HealthCheck(ctx)
				mu.Lock()
				checkResults[providerName] = err == nil
				mu.Unlock()

				if err != nil {
					log.Printf("[CHECK] %s: WARNING - Health check failed: %v", providerName, err)
				} else {
					log.Printf("[CHECK] %s: OK (Health check passed)", providerName)
				}
			}(name, p)
		}

		wg.Wait()

		// 标记通过健康检查的 provider 为可用
		for name, ok := range checkResults {
			if ok {
				providerManager.MarkAvailable(name)
			}
		}

		available := providerManager.ListAvailableProviders()
		log.Printf("[CHECK] Available GPU providers: %v", available)

		// 输出各 provider 的详细状态
		if vastaiHealth != nil && vastaiHealth.OK {
			log.Printf("[CHECK] Vast.ai GPU: OK (%s)", vastaiHealth.Message)
		} else if vastaiHealth != nil {
			log.Printf("[CHECK] Vast.ai GPU: WARNING - %s", vastaiHealth.Message)
		}

		if ppioHealth != nil && ppioHealth.OK {
			log.Printf("[CHECK] PPIO GPU: OK (%s)", ppioHealth.Message)
		} else if ppioHealth != nil {
			log.Printf("[CHECK] PPIO GPU: WARNING - %s", ppioHealth.Message)
		}
	} else {
		log.Printf("[CHECK] GPU Providers: SKIPPED (not configured)")
	}

	// 6. 测试创建任务（不实际执行）
	log.Println("Testing task creation...")
	testTask := &model.CreateTaskRequest{
		Name:    "startup-test-task",
		Image:   "python:3.11-slim",
		Command: "echo 'Startup test'",
	}

	if _, err := taskService.CreateTask(context.Background(), testTask); err != nil {
		log.Printf("[CHECK] Task Creation: WARNING (test task creation failed: %v)", err)
	} else {
		log.Printf("[CHECK] Task Creation: OK (test task created successfully)")
	}

	log.Println(strings.Repeat("=", 60))
	if allPassed {
		log.Println("All critical checks passed!")
	} else {
		log.Println("Some checks failed. Please review the configuration.")
	}
	log.Println(strings.Repeat("=", 60))
}

// VastaiHealth Vast.ai健康检查结果
type VastaiHealth struct {
	OK      bool
	Message string
}

// initVastai 初始化Vast.ai Provider
func initVastai(pm *provider.ProviderManager) *VastaiHealth {
	apiKey := getEnv("VASTAI_API_KEY", "")
	sshKey := getEnv("VASTAI_SSH_KEY", "")
	sshKeyID := getEnv("VASTAI_SSH_KEY_ID", "")

	log.Println("Checking Vast.ai configuration...")

	if apiKey == "" {
		return &VastaiHealth{
			OK:      false,
			Message: "VASTAI_API_KEY not configured",
		}
	}

	vastaiConfig := vastai.ProviderConfig{
		APIKey:      apiKey,
		SSHKey:      sshKey,
		SSHKeyID:    sshKeyID,
		DefaultGPU:  getEnv("VASTAI_DEFAULT_GPU", "RTX 4090"),
		DefaultDisk: 50,
		MaxPrice:    2.0,
	}

	vastaiProvider, err := vastai.NewProvider(vastaiConfig)
	if err != nil {
		return &VastaiHealth{
			OK:      false,
			Message: fmt.Sprintf("Vast.ai provider initialization failed: %v", err),
		}
	}

	// 注册到Provider管理器
	pm.RegisterProvider("vastai", vastaiProvider)

	log.Printf("Vast.ai configured: default_gpu=%s, max_price=$%.2f/hr", vastaiConfig.DefaultGPU, vastaiConfig.MaxPrice)

	// 测试连接 - 搜索一次GPU实例列表
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	offers, err := vastaiProvider.ListAvailableResources(ctx, &provider.ResourceRequest{
		GPUType: vastaiConfig.DefaultGPU,
	})
	if err != nil {
		return &VastaiHealth{
			OK:      false,
			Message: fmt.Sprintf("Vast.ai search test failed: %v", err),
		}
	}

	log.Printf("Vast.ai connection verified: found %d GPU offers", len(offers))

	return &VastaiHealth{
		OK:      true,
		Message: fmt.Sprintf("Vast.ai connection successful. Found %d GPU offers", len(offers)),
	}
}

// PPIOHealth PPIO健康检查结果
type PPIOHealth struct {
	OK      bool
	Message string
}

// initPPIO 初始化PPIO Provider
func initPPIO(pm *provider.ProviderManager) *PPIOHealth {
	apiKey := getEnv("PPIO_API_KEY", "")

	log.Println("Checking PPIO configuration...")

	if apiKey == "" {
		return &PPIOHealth{
			OK:      false,
			Message: "PPIO_API_KEY not configured",
		}
	}

	ppioConfig := ppio.ProviderConfig{
		APIKey:      apiKey,
		DefaultGPU:  getEnv("PPIO_DEFAULT_GPU", "RTX 4090"),
		DefaultDisk: 50,
		MaxPrice:    10.0,
	}

	ppioProvider, err := ppio.NewProvider(ppioConfig)
	if err != nil {
		return &PPIOHealth{
			OK:      false,
			Message: fmt.Sprintf("PPIO provider initialization failed: %v", err),
		}
	}

	// 注册到Provider管理器
	pm.RegisterProvider("ppio", ppioProvider)

	log.Printf("PPIO configured: default_gpu=%s, max_price=$%.2f/hr", ppioConfig.DefaultGPU, ppioConfig.MaxPrice)

	// 测试连接 - 获取GPU产品列表
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	products, err := ppioProvider.ListAvailableResources(ctx, &provider.ResourceRequest{
		GPUType: ppioConfig.DefaultGPU,
		MinGPUs: 1,
	})
	if err != nil {
		return &PPIOHealth{
			OK:      false,
			Message: fmt.Sprintf("PPIO product list test failed: %v", err),
		}
	}

	log.Printf("PPIO connection verified: found %d GPU products", len(products))

	return &PPIOHealth{
		OK:      true,
		Message: fmt.Sprintf("PPIO connection successful. Found %d GPU products", len(products)),
	}
}

// checkDocker 检查Docker是否运行
func checkDocker() bool {
	// 尝试检查Docker socket
	_, err := os.Stat("/var/run/docker.sock")
	if err != nil {
		return false
	}
	return true
}

// getEnv 获取环境变量，带默认值
// loadEnvFile 加载 .env 文件并设置环境变量（已存在的不覆盖）
func loadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(".env file not found at %s", path)
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// 去除可能的引号
		value = strings.Trim(value, `"'`)
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	log.Printf("Loaded environment variables from %s", path)
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
