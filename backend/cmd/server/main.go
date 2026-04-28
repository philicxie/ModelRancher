package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ml-platform/internal/api"
	"ml-platform/internal/executor"
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

	// 6. 运行启动自检
	runStartupChecks(taskService, cosClient, cosHealth)

	// 7. 初始化路由
	router := api.NewRouter(taskService, storageService, executorService)
	server := api.NewServer(router)

	// 8. 启动服务器
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
	if err := db.AutoMigrate(&model.Task{}, &model.TaskLog{}); err != nil {
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

// runStartupChecks 运行启动自检
func runStartupChecks(taskService *service.TaskService, cosClient *cosclient.Client, cosHealth COSHealth) {
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

	// 5. 测试创建任务（不实际执行）
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
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
