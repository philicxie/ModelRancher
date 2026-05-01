package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"ml-platform/internal/model"
	"ml-platform/internal/service"
)

// Server HTTP服务器
type Server struct {
	router *gin.Engine
	srv    *http.Server
}

// Router 路由
type Router struct {
	taskService     *service.TaskService
	storageService  *service.StorageService
	executor        *service.ExecutorService
	gpuService      *service.GPUService
	instanceService *service.InstanceService
}

// NewServer 创建服务器
func NewServer(router *Router) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 健康检查
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 任务管理
	tasks := r.Group("/api/v1/tasks")
	{
		tasks.POST("", router.createTask)
		tasks.GET("", router.listTasks)
		tasks.GET("/:id", router.getTask)
		tasks.DELETE("/:id", router.cancelTask)
		tasks.GET("/:id/logs", router.getTaskLogs)
	}

	// 存储管理
	storage := r.Group("/api/v1/storage")
	{
		storage.GET("/files", router.listFiles)
		storage.GET("/download-url", router.getDownloadURL)
	}

	// GPU实例管理
	gpu := r.Group("/api/v1/gpu")
	{
		gpu.GET("/instances", router.searchGPUInstances)
		gpu.GET("/providers", router.listGPUProviders)
	}

	// 实例租赁管理
	offers := r.Group("/api/v1/offers")
	{
		offers.POST("", router.createOffer)
		offers.GET("", router.listOffers)
		offers.GET("/:id", router.getOffer)
		offers.POST("/:id/stop", router.stopOffer)
		offers.POST("/:id/start", router.startOffer)
		offers.DELETE("/:id", router.destroyOffer)
	}

	return &Server{router: r}
}

// NewRouter 创建路由
func NewRouter(taskService *service.TaskService, storageService *service.StorageService, executor *service.ExecutorService, gpuService *service.GPUService, instanceService *service.InstanceService) *Router {
	return &Router{
		taskService:     taskService,
		storageService:  storageService,
		executor:        executor,
		gpuService:      gpuService,
		instanceService: instanceService,
	}
}

// Start 启动服务器
func (s *Server) Start(port string) error {
	s.srv = &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}
	return s.srv.ListenAndServe()
}

// Close 优雅关闭服务器
func (s *Server) Close() error {
	if s.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.srv.Shutdown(ctx)
	}
	return nil
}

// createTask 创建任务
func (r *Router) createTask(c *gin.Context) {
	var req model.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := r.taskService.CreateTask(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// listTasks 列出任务
func (r *Router) listTasks(c *gin.Context) {
	tasks, err := r.taskService.ListTasks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// getTask 获取任务
func (r *Router) getTask(c *gin.Context) {
	taskID := c.Param("id")
	task, err := r.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

// cancelTask 取消任务
func (r *Router) cancelTask(c *gin.Context) {
	taskID := c.Param("id")
	if err := r.taskService.CancelTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task cancelled"})
}

// getTaskLogs 获取任务日志
func (r *Router) getTaskLogs(c *gin.Context) {
	taskID := c.Param("id")

	// 生成客户端ID
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = generateClientID()
	}

	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 订阅日志
	logChan := r.taskService.SubscribeLogs(taskID, clientID)
	defer r.taskService.UnsubscribeLogs(taskID, clientID)

	// 发送日志流
	clientGone := c.Request.Context().Done()
	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				return
			}
			c.SSEvent("log", log)
			c.Writer.Flush()
		case <-clientGone:
			return
		}
	}
}

// listFiles 列出文件
func (r *Router) listFiles(c *gin.Context) {
	prefix := c.Query("prefix")

	objects, err := r.storageService.ListObjects(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, objects)
}

// getDownloadURL 获取下载URL
func (r *Router) getDownloadURL(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	url, err := r.storageService.GetPresignedURL(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// listGPUProviders 列出可用的GPU Provider
func (r *Router) listGPUProviders(c *gin.Context) {
	providers := r.gpuService.ListProviders(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

// searchGPUInstances 搜索GPU实例
func (r *Router) searchGPUInstances(c *gin.Context) {
	var req service.SearchGPUInstancesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instances, err := r.gpuService.SearchGPUInstances(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"total":     len(instances),
	})
}

// createOffer 创建租赁
func (r *Router) createOffer(c *gin.Context) {
	var req model.CreateOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance, err := r.instanceService.CreateOffer(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, instance)
}

// listOffers 列出用户的租赁实例
func (r *Router) listOffers(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	instances, err := r.instanceService.ListUserInstances(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instances": instances,
		"total":     len(instances),
	})
}

// getOffer 获取租赁详情
func (r *Router) getOffer(c *gin.Context) {
	id := c.Param("id")
	instance, err := r.instanceService.GetInstance(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, instance)
}

// stopOffer 停止实例
func (r *Router) stopOffer(c *gin.Context) {
	id := c.Param("id")
	if err := r.instanceService.StopInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Instance stopped"})
}

// startOffer 启动实例
func (r *Router) startOffer(c *gin.Context) {
	id := c.Param("id")
	if err := r.instanceService.StartInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Instance starting"})
}

// destroyOffer 销毁实例
func (r *Router) destroyOffer(c *gin.Context) {
	id := c.Param("id")
	if err := r.instanceService.DestroyInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Instance destroyed"})
}

var clientIDCounter int
var clientIDMu sync.Mutex

func generateClientID() string {
	clientIDMu.Lock()
	defer clientIDMu.Unlock()
	clientIDCounter++
	return "client-" + string(rune('0'+clientIDCounter%10))
}
