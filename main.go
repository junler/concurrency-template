package main

import (
	"log"

	"concurrency-web-app/backend/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建Gin路由器
	r := gin.Default()

	// 配置CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// 静态文件服务
	r.Static("/static", "./frontend")
	r.StaticFile("/", "./frontend/index.html")

	// 创建处理器
	batchHandler := handlers.NewBatchHandler()

	// 设置路由
	batchHandler.SetupRoutes(r)

	// 启动服务器
	log.Println("服务器启动在端口 :8080")
	log.Println("前端访问: http://localhost:8080")
	log.Println("API文档: http://localhost:8080/api/health")

	if err := r.Run(":8080"); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}
