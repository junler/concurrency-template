# 并发处理演示系统

这是一个基于Go语言的并发处理演示系统，展示了三种实际应用场景的并发处理：批量订单处理、API批量调用和文件批量处理。

## 根据一段代码生成的提示词

https://mp.weixin.qq.com/s/eAsFyugeVlXrmV1AwYg6eQ
整理这里面的代码，给我工作中实际的场景demo

对接前端页面,用tailwindcss和bootstrap的最新版本实现web页面，后端使用gin(数据库使用gorm+sqlite)

## 项目结构

```
web-app/
├── backend/                 # 后端Go代码
│   ├── main.go             # 主程序入口
│   ├── go.mod              # Go模块配置
│   ├── models/             # 数据模型
│   │   └── models.go       # 数据库模型定义
│   ├── services/           # 业务逻辑服务
│   │   └── batch_service.go # 批量处理服务
│   └── handlers/           # HTTP处理器
│       └── batch_handler.go # 批量处理API处理器
├── frontend/               # 前端代码
│   └── index.html          # 单页面应用
└── uploads/                # 文件上传目录
```

## 启动运行

go run .
air

## 技术栈

### 后端
- **Go 1.21+**: 主要编程语言
- **Gin**: Web框架，提供HTTP路由和中间件
- **GORM**: ORM框架，简化数据库操作
- **SQLite**: 轻量级数据库，存储处理记录

### 前端
- **HTML5 + JavaScript**: 基础技术
- **Bootstrap 5.3.2**: UI组件库
- **Tailwind CSS**: 实用工具CSS框架
- **Chart.js**: 数据可视化图表库
- **Font Awesome**: 图标库

## 功能特性

### 1. 批量订单处理
- 🛒 **场景**: 电商系统批量处理订单
- ⚡ **并发处理**: 可配置最大并发数
- 📊 **实时统计**: 成功/失败计数，处理时间
- 🔄 **容错机制**: 支持单个订单失败但整体继续
- 📈 **性能监控**: 可视化处理性能

### 2. API批量调用
- 🌐 **场景**: 同时调用多个外部API服务
- 🚀 **高并发**: 支持同时调用多个HTTP API
- ⏱️ **超时控制**: 可配置请求超时时间
- 📋 **响应记录**: 记录状态码、响应体等详细信息
- 🔁 **重试机制**: 支持失败重试

### 3. 文件批量处理
- 📁 **场景**: 批量处理上传的文件
- 🔧 **多种操作**: 文件信息获取、复制、压缩等
- 📤 **文件上传**: 支持多文件同时上传
- 🗂️ **文件管理**: 查看已上传文件列表
- ⚙️ **处理类型**: 可选择不同的处理方式

## 核心并发特性

### 并发控制
```go
// 信号量控制并发数
semaphore := make(chan struct{}, maxConcurrency)

// 协程处理任务
go func(task Task) {
    defer wg.Done()
    
    // 获取信号量
    semaphore <- struct{}{}
    defer func() { <-semaphore }()
    
    // 执行任务...
}(task)
```

### 超时处理
```go
// 创建超时上下文
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()

// 检查超时
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
    // 继续执行...
}
```

### 结果收集
```go
// 使用通道收集结果
resultCh := make(chan TaskResult, totalTasks)

// 等待所有任务完成
go func() {
    wg.Wait()
    close(resultCh)
}()

// 收集结果并排序
for result := range resultCh {
    results = append(results, result)
}
sort.Slice(results, func(i, j int) bool {
    return results[i].ID < results[j].ID
})
```

## 快速开始

### 1. 安装依赖

```bash
cd backend
go mod tidy
```

### 2. 启动服务器

```bash
go run main.go
```

### 3. 访问应用

打开浏览器访问: `http://localhost:8080`

## API接口

### 订单处理
- `POST /api/orders/generate` - 生成测试订单
- `POST /api/orders/batch-process` - 批量处理订单

### API调用
- `POST /api/api-calls/generate` - 生成API调用列表
- `POST /api/api-calls/batch-call` - 批量调用API

### 文件处理
- `POST /api/files/upload` - 上传文件
- `GET /api/files/list` - 获取文件列表
- `POST /api/files/batch-process` - 批量处理文件

### 健康检查
- `GET /api/health` - 服务健康检查

## 配置说明

### 并发配置
```go
OrderService: &services.OrderProcessService{
    MaxConcurrency: 10,        // 最大并发数
    Timeout:        30 * time.Second, // 超时时间
}
```

### 数据库配置
- 使用SQLite数据库，文件名：`concurrency_app.db`
- 自动创建表结构
- 支持数据持久化

## 性能优化

### 1. 并发控制
- 使用信号量限制并发数，避免资源耗尽
- 协程池复用，减少创建销毁开销

### 2. 内存管理
- 使用缓冲通道避免协程阻塞
- 及时释放资源，避免内存泄漏

### 3. 错误处理
- 单个任务失败不影响整体
- 支持panic恢复机制
- 详细的错误信息记录

## 扩展建议

### 1. 数据库优化
- 使用MySQL或PostgreSQL替换SQLite
- 添加连接池配置
- 实现读写分离

### 2. 缓存机制
- 集成Redis缓存
- 实现结果缓存
- 添加分布式锁

### 3. 监控告警
- 集成Prometheus监控
- 添加性能指标收集
- 实现告警机制

### 4. 分布式部署
- 支持微服务架构
- 使用消息队列解耦
- 实现服务发现

## 注意事项

1. **并发数控制**: 根据服务器性能调整最大并发数
2. **超时设置**: 合理设置超时时间，避免长时间等待
3. **错误处理**: 确保所有异常都被正确处理
4. **资源清理**: 及时释放文件句柄和网络连接
5. **安全考虑**: 文件上传需要验证文件类型和大小

## 学习要点

这个项目演示了Go语言并发编程的核心概念：

1. **Goroutines**: 轻量级协程的使用
2. **Channels**: 协程间通信机制
3. **WaitGroup**: 等待协程组完成
4. **Context**: 超时和取消控制
5. **Select**: 多路复用选择
6. **Mutex**: 互斥锁保护共享资源

通过这个实际项目，可以深入理解Go语言在高并发场景下的优势和最佳实践。
