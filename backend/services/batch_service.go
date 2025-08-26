package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// TaskResult 通用任务结果
type TaskResult struct {
	ID       int         `json:"id"`
	Success  bool        `json:"success"`
	Data     interface{} `json:"data"`
	Error    string      `json:"error,omitempty"`
	Duration int64       `json:"duration"` // 毫秒
}

// BatchResult 批量处理结果
type BatchResult struct {
	TotalTasks   int          `json:"total_tasks"`
	SuccessTasks int          `json:"success_tasks"`
	FailedTasks  int          `json:"failed_tasks"`
	Results      []TaskResult `json:"results"`
	Duration     int64        `json:"duration"` // 毫秒
}

// OrderProcessService 订单处理服务
type OrderProcessService struct {
	MaxConcurrency int
	Timeout        time.Duration
}

// OrderTask 订单处理任务
type OrderTask struct {
	ID          int     `json:"id"`
	CustomerID  string  `json:"customer_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

// ProcessOrder 处理单个订单
func (s *OrderProcessService) ProcessOrder(order OrderTask) (interface{}, error) {
	// 模拟订单处理时间
	processingTime := time.Duration(100+order.ID*10) * time.Millisecond
	time.Sleep(processingTime)

	// 模拟某些订单处理失败
	if order.ID%7 == 0 {
		return nil, fmt.Errorf("订单 %d 库存不足", order.ID)
	}

	// 计算总价
	totalPrice := order.Price * float64(order.Quantity)

	return map[string]interface{}{
		"order_id":     order.ID,
		"customer_id":  order.CustomerID,
		"product_name": order.ProductName,
		"quantity":     order.Quantity,
		"unit_price":   order.Price,
		"total_price":  totalPrice,
		"status":       "processed",
		"processed_at": time.Now(),
	}, nil
}

// BatchProcessOrders 批量处理订单
func (s *OrderProcessService) BatchProcessOrders(ctx context.Context, orders []OrderTask) *BatchResult {
	startTime := time.Now()
	totalTasks := len(orders)

	if totalTasks == 0 {
		return &BatchResult{
			TotalTasks: 0,
			Results:    []TaskResult{},
			Duration:   time.Since(startTime).Milliseconds(),
		}
	}

	resultCh := make(chan TaskResult, totalTasks)
	var wg sync.WaitGroup

	// 限制并发数
	semaphore := make(chan struct{}, s.MaxConcurrency)

	for i, order := range orders {
		wg.Add(1)
		go func(index int, task OrderTask) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			taskStart := time.Now()

			// 检查超时
			select {
			case <-ctx.Done():
				resultCh <- TaskResult{
					ID:       index,
					Success:  false,
					Error:    "任务超时",
					Duration: time.Since(taskStart).Milliseconds(),
				}
				return
			default:
			}

			// 处理订单
			data, err := s.ProcessOrder(task)

			result := TaskResult{
				ID:       index,
				Success:  err == nil,
				Data:     data,
				Duration: time.Since(taskStart).Milliseconds(),
			}

			if err != nil {
				result.Error = err.Error()
			}

			resultCh <- result
		}(i, order)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果
	var results []TaskResult
	successCount := 0

	timeout := time.NewTimer(s.Timeout)
	defer timeout.Stop()

	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				goto DONE
			}
			results = append(results, result)
			if result.Success {
				successCount++
			}
		case <-timeout.C:
			goto DONE
		case <-ctx.Done():
			goto DONE
		}
	}

DONE:
	// 按ID排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return &BatchResult{
		TotalTasks:   totalTasks,
		SuccessTasks: successCount,
		FailedTasks:  totalTasks - successCount,
		Results:      results,
		Duration:     time.Since(startTime).Milliseconds(),
	}
}

// APICallService API调用服务
type APICallService struct {
	MaxConcurrency int
	Timeout        time.Duration
	Client         *http.Client
}

// APICallTask API调用任务
type APICallTask struct {
	ID      int               `json:"id"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// CallAPI 调用单个API
func (s *APICallService) CallAPI(task APICallTask) (interface{}, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	var bodyReader io.Reader
	if task.Body != "" {
		bodyReader = strings.NewReader(task.Body)
	}

	req, err := http.NewRequest(task.Method, task.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	for key, value := range task.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	return map[string]interface{}{
		"url":           task.URL,
		"method":        task.Method,
		"status_code":   resp.StatusCode,
		"response_body": string(body),
		"headers":       resp.Header,
	}, nil
}

// BatchCallAPIs 批量调用API
func (s *APICallService) BatchCallAPIs(ctx context.Context, tasks []APICallTask) *BatchResult {
	startTime := time.Now()
	totalTasks := len(tasks)

	if totalTasks == 0 {
		return &BatchResult{
			TotalTasks: 0,
			Results:    []TaskResult{},
			Duration:   time.Since(startTime).Milliseconds(),
		}
	}

	resultCh := make(chan TaskResult, totalTasks)
	var wg sync.WaitGroup

	// 限制并发数
	semaphore := make(chan struct{}, s.MaxConcurrency)

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, apiTask APICallTask) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			taskStart := time.Now()

			// 检查超时
			select {
			case <-ctx.Done():
				resultCh <- TaskResult{
					ID:       index,
					Success:  false,
					Error:    "任务超时",
					Duration: time.Since(taskStart).Milliseconds(),
				}
				return
			default:
			}

			// 调用API
			data, err := s.CallAPI(apiTask)

			result := TaskResult{
				ID:       index,
				Success:  err == nil,
				Data:     data,
				Duration: time.Since(taskStart).Milliseconds(),
			}

			if err != nil {
				result.Error = err.Error()
			}

			resultCh <- result
		}(i, task)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果
	var results []TaskResult
	successCount := 0

	timeout := time.NewTimer(s.Timeout)
	defer timeout.Stop()

	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				goto DONE
			}
			results = append(results, result)
			if result.Success {
				successCount++
			}
		case <-timeout.C:
			goto DONE
		case <-ctx.Done():
			goto DONE
		}
	}

DONE:
	// 按ID排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return &BatchResult{
		TotalTasks:   totalTasks,
		SuccessTasks: successCount,
		FailedTasks:  totalTasks - successCount,
		Results:      results,
		Duration:     time.Since(startTime).Milliseconds(),
	}
}

// FileProcessService 文件处理服务
type FileProcessService struct {
	MaxConcurrency int
	Timeout        time.Duration
	UploadDir      string
}

// FileTask 文件处理任务
type FileTask struct {
	ID          int    `json:"id"`
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	ProcessType string `json:"process_type"` // info, copy, move, compress
}

// ProcessFile 处理单个文件
func (s *FileProcessService) ProcessFile(task FileTask) (interface{}, error) {
	// 模拟文件处理时间
	time.Sleep(time.Duration(200+task.ID*50) * time.Millisecond)

	// 获取文件信息
	fileInfo, err := os.Stat(task.FilePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	result := map[string]interface{}{
		"file_path":    task.FilePath,
		"file_name":    task.FileName,
		"file_size":    fileInfo.Size(),
		"process_type": task.ProcessType,
		"processed_at": time.Now(),
	}

	switch task.ProcessType {
	case "info":
		result["info"] = map[string]interface{}{
			"size":      fileInfo.Size(),
			"mode":      fileInfo.Mode().String(),
			"mod_time":  fileInfo.ModTime(),
			"is_dir":    fileInfo.IsDir(),
			"extension": filepath.Ext(task.FileName),
		}
	case "copy":
		// 模拟文件复制
		copyPath := filepath.Join(s.UploadDir, "copy_"+task.FileName)
		err := s.copyFile(task.FilePath, copyPath)
		if err != nil {
			return nil, fmt.Errorf("复制文件失败: %v", err)
		}
		result["copy_path"] = copyPath
	case "compress":
		// 模拟文件压缩（这里只是示例，实际项目中需要真正的压缩逻辑）
		result["compressed_size"] = fileInfo.Size() / 2 // 模拟压缩后大小
		result["compression_ratio"] = "50%"
	default:
		return nil, fmt.Errorf("不支持的处理类型: %s", task.ProcessType)
	}

	return result, nil
}

// copyFile 复制文件
func (s *FileProcessService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// BatchProcessFiles 批量处理文件
func (s *FileProcessService) BatchProcessFiles(ctx context.Context, tasks []FileTask) *BatchResult {
	startTime := time.Now()
	totalTasks := len(tasks)

	if totalTasks == 0 {
		return &BatchResult{
			TotalTasks: 0,
			Results:    []TaskResult{},
			Duration:   time.Since(startTime).Milliseconds(),
		}
	}

	resultCh := make(chan TaskResult, totalTasks)
	var wg sync.WaitGroup

	// 限制并发数
	semaphore := make(chan struct{}, s.MaxConcurrency)

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, fileTask FileTask) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			taskStart := time.Now()

			// 检查超时
			select {
			case <-ctx.Done():
				resultCh <- TaskResult{
					ID:       index,
					Success:  false,
					Error:    "任务超时",
					Duration: time.Since(taskStart).Milliseconds(),
				}
				return
			default:
			}

			// 处理文件
			data, err := s.ProcessFile(fileTask)

			result := TaskResult{
				ID:       index,
				Success:  err == nil,
				Data:     data,
				Duration: time.Since(taskStart).Milliseconds(),
			}

			if err != nil {
				result.Error = err.Error()
			}

			resultCh <- result
		}(i, task)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果
	var results []TaskResult
	successCount := 0

	timeout := time.NewTimer(s.Timeout)
	defer timeout.Stop()

	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				goto DONE
			}
			results = append(results, result)
			if result.Success {
				successCount++
			}
		case <-timeout.C:
			goto DONE
		case <-ctx.Done():
			goto DONE
		}
	}

DONE:
	// 按ID排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})

	return &BatchResult{
		TotalTasks:   totalTasks,
		SuccessTasks: successCount,
		FailedTasks:  totalTasks - successCount,
		Results:      results,
		Duration:     time.Since(startTime).Milliseconds(),
	}
}
