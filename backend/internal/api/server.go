package api

import (
	"context"
	"net/http"
	"strconv"
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
	imageService    *service.ImageService
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
		tasks.DELETE("/:id", router.deleteTask)
		tasks.POST("/:id/cancel", router.cancelTask)
		tasks.GET("/:id/logs", router.getTaskLogs)
	}

	// 存储管理
	storage := r.Group("/api/v1/storage")
	{
		storage.GET("/buckets", router.listBuckets)
		storage.POST("/buckets", router.createBucket)
		storage.DELETE("/buckets/:id", router.deleteBucket)
		storage.GET("/files", router.listFiles)
		storage.GET("/download-url", router.getDownloadURL)
		storage.POST("/upload", router.uploadFile)
		storage.DELETE("/files", router.deleteFile)
		storage.POST("/folders", router.createFolder)
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
		offers.GET("/:id/metrics", router.getOfferMetrics)
		offers.POST("/:id/stop", router.stopOffer)
		offers.POST("/:id/start", router.startOffer)
		offers.DELETE("/:id", router.destroyOffer)
	}

	// 历史工单
	workOrders := r.Group("/api/v1/work-orders")
	{
		workOrders.GET("", router.listWorkOrders)
	}

	// 镜像管理
	images := r.Group("/api/v1/images")
	{
		images.GET("/search", router.searchImages)
		images.GET("/private", router.listPrivateImages)
		images.GET("/favorites", router.listFavorites)
		images.POST("/favorites", router.addFavorite)
		images.DELETE("/favorites", router.removeFavorite)
	}

	return &Server{router: r}
}

// NewRouter 创建路由
func NewRouter(taskService *service.TaskService, storageService *service.StorageService, executor *service.ExecutorService, gpuService *service.GPUService, instanceService *service.InstanceService, imageService *service.ImageService) *Router {
	return &Router{
		taskService:     taskService,
		storageService:  storageService,
		executor:        executor,
		gpuService:      gpuService,
		instanceService: instanceService,
		imageService:    imageService,
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

// deleteTask 删除任务
func (r *Router) deleteTask(c *gin.Context) {
	taskID := c.Param("id")
	if err := r.taskService.DeleteTask(c.Request.Context(), taskID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
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

// listBuckets 列出桶
func (r *Router) listBuckets(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	buckets, err := r.storageService.ListBuckets(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"buckets": buckets,
		"total":   len(buckets),
	})
}

// createBucket 创建桶
func (r *Router) createBucket(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		IsPublic bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	bucket, err := r.storageService.CreateBucket(c.Request.Context(), userID, req.Name, req.IsPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bucket)
}

// deleteBucket 删除桶
func (r *Router) deleteBucket(c *gin.Context) {
	bucketID := c.Param("id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	if err := r.storageService.DeleteBucket(c.Request.Context(), bucketID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bucket deleted"})
}

// listFiles 列出文件
func (r *Router) listFiles(c *gin.Context) {
	bucketName := c.Query("bucket")
	if bucketName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required"})
		return
	}
	prefix := c.Query("prefix")

	objects, err := r.storageService.ListObjects(c.Request.Context(), bucketName, prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, objects)
}

// getDownloadURL 获取下载URL
func (r *Router) getDownloadURL(c *gin.Context) {
	bucketName := c.Query("bucket")
	if bucketName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required"})
		return
	}
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	url, err := r.storageService.GetPresignedURL(c.Request.Context(), bucketName, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// uploadFile 上传文件到COS
func (r *Router) uploadFile(c *gin.Context) {
	bucketName := c.PostForm("bucket")
	if bucketName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	prefix := c.PostForm("prefix")
	key := prefix + header.Filename

	if err := r.storageService.UploadFromReader(c.Request.Context(), bucketName, key, file, header.Size); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "uploaded",
		"key":    key,
	})
}

// deleteFile 删除COS文件
func (r *Router) deleteFile(c *gin.Context) {
	bucketName := c.Query("bucket")
	if bucketName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bucket is required"})
		return
	}
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	if err := r.storageService.DeleteObject(c.Request.Context(), bucketName, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// createFolder 创建文件夹
func (r *Router) createFolder(c *gin.Context) {
	var req struct {
		Bucket string `json:"bucket" binding:"required"`
		Prefix string `json:"prefix"`
		Name   string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := req.Prefix + req.Name + "/"
	if err := r.storageService.CreateFolder(c.Request.Context(), req.Bucket, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "created",
		"key":    key,
	})
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

// getOfferMetrics 获取实例监控指标
func (r *Router) getOfferMetrics(c *gin.Context) {
	id := c.Param("id")
	startTime := int64(0)
	endTime := int64(0)
	if s := c.Query("start_time"); s != "" {
		startTime, _ = strconv.ParseInt(s, 10, 64)
	}
	if e := c.Query("end_time"); e != "" {
		endTime, _ = strconv.ParseInt(e, 10, 64)
	}

	metrics, err := r.instanceService.GetInstanceMetrics(c.Request.Context(), id, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
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

// listWorkOrders 列出历史工单
func (r *Router) listWorkOrders(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	orders, err := r.instanceService.ListWorkOrders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  len(orders),
	})
}

var clientIDCounter int
var clientIDMu sync.Mutex

func generateClientID() string {
	clientIDMu.Lock()
	defer clientIDMu.Unlock()
	clientIDCounter++
	return "client-" + string(rune('0'+clientIDCounter%10))
}

// ============================================================================
// 镜像管理
// ============================================================================

// searchImages 搜索 Docker Hub 公开镜像
func (r *Router) searchImages(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	result, err := r.imageService.SearchPublicImages(c.Request.Context(), query, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查每个结果是否已被当前用户收藏
	favorites, _ := r.imageService.ListFavorites(c.Request.Context(), userID)
	favMap := make(map[string]bool)
	for _, f := range favorites {
		favMap[f.ImageName] = true
	}

	images := make([]gin.H, 0, len(result.Results))
	for _, item := range result.Results {
		images = append(images, gin.H{
			"name":         item.Name,
			"description":  item.Description,
			"star_count":   item.StarCount,
			"is_official":  item.IsOfficial,
			"is_favorited": favMap[item.Name],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   result.Count,
		"page":    result.Page,
		"images":  images,
	})
}

// listPrivateImages 列出私有镜像
func (r *Router) listPrivateImages(c *gin.Context) {
	provider := c.Query("provider")
	images, err := r.imageService.ListPrivateImages(c.Request.Context(), provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": images})
}

// listFavorites 列出用户收藏的镜像
func (r *Router) listFavorites(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "default"
	}

	favorites, err := r.imageService.ListFavorites(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorites": favorites})
}

// addFavorite 添加收藏
func (r *Router) addFavorite(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id"`
		ImageName   string `json:"image_name" binding:"required"`
		Description string `json:"description"`
		StarCount   int    `json:"star_count"`
		IsOfficial  bool   `json:"is_official"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.UserID == "" {
		req.UserID = "default"
	}

	fav, err := r.imageService.AddFavorite(c.Request.Context(), req.UserID, req.ImageName, req.Description, req.StarCount, req.IsOfficial)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, fav)
}

// removeFavorite 取消收藏
func (r *Router) removeFavorite(c *gin.Context) {
	userID := c.Query("user_id")
	imageName := c.Query("image_name")
	if userID == "" {
		userID = "default"
	}
	if imageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_name is required"})
		return
	}

	if err := r.imageService.RemoveFavorite(c.Request.Context(), userID, imageName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed from favorites"})
}
