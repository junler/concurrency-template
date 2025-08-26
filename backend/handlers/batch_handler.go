package handlers

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"concurrency-web-app/backend/services"

	"github.com/gin-gonic/gin"
)

// BatchHandler 批量处理控制器
type BatchHandler struct {
	OrderService *services.OrderProcessService
	APIService   *services.APICallService
	FileService  *services.FileProcessService
}

// NewBatchHandler 创建新的批量处理控制器
func NewBatchHandler() *BatchHandler {
	return &BatchHandler{
		OrderService: &services.OrderProcessService{
			MaxConcurrency: 10,
			Timeout:        30 * time.Second,
		},
		APIService: &services.APICallService{
			MaxConcurrency: 5,
			Timeout:        60 * time.Second,
			Client:         &http.Client{Timeout: 10 * time.Second},
		},
		FileService: &services.FileProcessService{
			MaxConcurrency: 3,
			Timeout:        120 * time.Second,
			UploadDir:      "./uploads",
		},
	}
}

// BatchProcessOrdersRequest 批量处理订单请求
type BatchProcessOrdersRequest struct {
	Orders []services.OrderTask `json:"orders" binding:"required"`
}

// BatchProcessOrders 批量处理订单
func (h *BatchHandler) BatchProcessOrders(c *gin.Context) {
	var req BatchProcessOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), h.OrderService.Timeout)
	defer cancel()

	// 执行批量处理
	result := h.OrderService.BatchProcessOrders(ctx, req.Orders)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量订单处理完成",
		"data":    result,
	})
}

// GenerateOrdersRequest 生成订单请求
type GenerateOrdersRequest struct {
	Count int `json:"count" binding:"required,min=1,max=1000"`
}

// GenerateOrders 生成测试订单
func (h *BatchHandler) GenerateOrders(c *gin.Context) {
	var req GenerateOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	orders := make([]services.OrderTask, req.Count)
	products := []string{"iPhone 15", "MacBook Pro", "iPad Air", "Apple Watch", "AirPods Pro"}

	for i := 0; i < req.Count; i++ {
		orders[i] = services.OrderTask{
			ID:          i + 1,
			CustomerID:  fmt.Sprintf("CUST_%04d", i+1),
			ProductName: products[i%len(products)],
			Quantity:    (i % 5) + 1,
			Price:       float64(100 + (i%10)*50),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单生成成功",
		"data":    orders,
	})
}

// BatchCallAPIsRequest 批量API调用请求
type BatchCallAPIsRequest struct {
	APIs []services.APICallTask `json:"apis" binding:"required"`
}

// BatchCallAPIs 批量调用API
func (h *BatchHandler) BatchCallAPIs(c *gin.Context) {
	var req BatchCallAPIsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), h.APIService.Timeout)
	defer cancel()

	// 执行批量调用
	result := h.APIService.BatchCallAPIs(ctx, req.APIs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量API调用完成",
		"data":    result,
	})
}

// GenerateAPICallsRequest 生成API调用请求
type GenerateAPICallsRequest struct {
	Count int `json:"count" binding:"required,min=1,max=50"`
}

// GenerateAPICalls 生成测试API调用
func (h *BatchHandler) GenerateAPICalls(c *gin.Context) {
	var req GenerateAPICallsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	apis := make([]services.APICallTask, req.Count)
	testAPIs := []string{
		"https://jsonplaceholder.typicode.com/posts",
		"https://httpbin.org/get",
		"https://api.github.com/users/octocat",
		"https://httpbin.org/delay/1",
		"https://httpbin.org/status/200",
	}

	for i := 0; i < req.Count; i++ {
		apis[i] = services.APICallTask{
			ID:     i + 1,
			URL:    testAPIs[i%len(testAPIs)],
			Method: "GET",
			Headers: map[string]string{
				"User-Agent": "ConcurrencyApp/1.0",
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API调用列表生成成功",
		"data":    apis,
	})
}

// UploadFiles 文件上传
func (h *BatchHandler) UploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析文件失败: " + err.Error()})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有选择文件"})
		return
	}

	// 确保上传目录存在
	uploadDir := h.FileService.UploadDir
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败: " + err.Error()})
		return
	}

	var uploadedFiles []map[string]interface{}

	for _, file := range files {
		// 生成唯一文件名
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
		filePath := filepath.Join(uploadDir, filename)

		// 保存文件
		if err := h.saveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败: " + err.Error()})
			return
		}

		uploadedFiles = append(uploadedFiles, map[string]interface{}{
			"original_name": file.Filename,
			"saved_name":    filename,
			"file_path":     filePath,
			"size":          file.Size,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "文件上传成功",
		"data":    uploadedFiles,
	})
}

// saveUploadedFile 保存上传的文件
func (h *BatchHandler) saveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// BatchProcessFilesRequest 批量处理文件请求
type BatchProcessFilesRequest struct {
	Files []services.FileTask `json:"files" binding:"required"`
}

// BatchProcessFiles 批量处理文件
func (h *BatchHandler) BatchProcessFiles(c *gin.Context) {
	var req BatchProcessFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), h.FileService.Timeout)
	defer cancel()

	// 执行批量处理
	result := h.FileService.BatchProcessFiles(ctx, req.Files)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量文件处理完成",
		"data":    result,
	})
}

// ListUploadedFiles 列出已上传的文件
func (h *BatchHandler) ListUploadedFiles(c *gin.Context) {
	uploadDir := h.FileService.UploadDir

	// 读取目录
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取目录失败: " + err.Error()})
		return
	}

	var files []map[string]interface{}
	for i, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, map[string]interface{}{
				"id":        i + 1,
				"file_name": entry.Name(),
				"file_path": filepath.Join(uploadDir, entry.Name()),
				"size":      info.Size(),
				"mod_time":  info.ModTime(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "文件列表获取成功",
		"data":    files,
	})
}

// SetupRoutes 设置路由
func (h *BatchHandler) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 订单处理相关路由
		orders := api.Group("/orders")
		{
			orders.POST("/generate", h.GenerateOrders)
			orders.POST("/batch-process", h.BatchProcessOrders)
		}

		// API调用相关路由
		apiCalls := api.Group("/api-calls")
		{
			apiCalls.POST("/generate", h.GenerateAPICalls)
			apiCalls.POST("/batch-call", h.BatchCallAPIs)
		}

		// 文件处理相关路由
		files := api.Group("/files")
		{
			files.POST("/upload", h.UploadFiles)
			files.GET("/list", h.ListUploadedFiles)
			files.POST("/batch-process", h.BatchProcessFiles)
		}

		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "ok",
				"timestamp": time.Now(),
				"message":   "Concurrency Web App is running",
			})
		})
	}
}
