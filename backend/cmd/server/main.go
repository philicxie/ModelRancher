package main

import (
	"log"
	"os"

	"ml-platform/internal/api"
	"ml-platform/internal/executor"
	"ml-platform/internal/service"
	"ml-platform/pkg/cosclient"
)

func main() {
	// 配置日志
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 从环境变量读取配置
	cosSecretID := os.Getenv("TENCENT_CLOUD_SECRET_ID")
	cosSecretKey := os.Getenv("TENCENT_CLOUD_SECRET_KEY")
	cosBucket := os.Getenv("TENCENT_CLOUD_COS_BUCKET")
	cosRegion := os.Getenv("TENCENT_CLOUD_COS_REGION")

	log.Printf("Starting ML Training Platform...")
	log.Printf("COS Bucket: %s, Region: %s", cosBucket, cosRegion)

	// 初始化COS客户端（如果配置不完整会返回错误，但不影响启动）
	var cosClient *cosclient.Client
	var err error
	if cosSecretID != "" && cosSecretKey != "" && cosBucket != "" && cosRegion != "" {
		cosClient, err = cosclient.NewCOSClient(cosSecretID, cosSecretKey, cosBucket, cosRegion)
		if err != nil {
			log.Printf("Warning: Failed to initialize COS client: %v", err)
			log.Printf("Storage operations will fail until COS is properly configured")
		}
	} else {
		log.Printf("Warning: COS configuration not found in environment variables")
		log.Printf("Please set TENCENT_CLOUD_SECRET_ID, TENCENT_CLOUD_SECRET_KEY, TENCENT_CLOUD_COS_BUCKET, TENCENT_CLOUD_COS_REGION")
	}

	// 初始化执行器
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/app/data"
	}
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "/app/output"
	}
	execEngine := executor.NewExecutor(dataDir, outputDir)

	// 初始化服务
	taskService := service.NewTaskService(execEngine, cosClient)
	storageService := service.NewStorageService(cosClient)
	executorService := service.NewExecutorService(execEngine)

	// 初始化路由
	router := api.NewRouter(taskService, storageService, executorService)
	server := api.NewServer(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := server.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}