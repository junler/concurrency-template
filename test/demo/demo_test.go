package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"testing"
	"time"
)

// Order 订单信息
type Order struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// OrderWithSeq 带序号的订单结果
type OrderWithSeq struct {
	Seq   int   `json:"seq"`
	Order Order `json:"order"`
	Error error `json:"error,omitempty"`
}

// BySeq 实现sort.Interface，用于按序号排序
type BySeq []OrderWithSeq

func (a BySeq) Len() int           { return len(a) }
func (a BySeq) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySeq) Less(i, j int) bool { return a[i].Seq < a[j].Seq }

// Task 任务接口
type Task interface {
	Execute() (Order, error)
	GetID() int
}

// OrderTask 订单处理任务
type OrderTask struct {
	ID          int
	Name        string
	Duration    time.Duration // 模拟任务执行时间
	ShouldError bool          // 是否模拟错误
	ShouldPanic bool          // 是否模拟panic
}

func (t *OrderTask) Execute() (Order, error) {
	// 模拟任务执行时间
	time.Sleep(t.Duration)

	// 模拟panic情况
	if t.ShouldPanic {
		panic(fmt.Sprintf("task %d panicked", t.ID))
	}

	// 模拟错误情况
	if t.ShouldError {
		return Order{}, fmt.Errorf("task %d failed", t.ID)
	}

	// 正常执行
	return Order{
		ID:   t.ID,
		Name: fmt.Sprintf("Order_%s_%d", t.Name, t.ID),
	}, nil
}

func (t *OrderTask) GetID() int {
	return t.ID
}

// BatchTaskProcessor 批量任务处理器
type BatchTaskProcessor struct {
	MaxConcurrency int
	Timeout        time.Duration
	KeepOrder      bool // 是否保持顺序
}

// ProcessTasksResult 批量处理结果
type ProcessTasksResult struct {
	Orders       []Order       `json:"orders"`
	Errors       []error       `json:"errors"`
	Duration     time.Duration `json:"duration"`
	SuccessCount int           `json:"success_count"`
	ErrorCount   int           `json:"error_count"`
}

// ProcessTasks 批量处理任务（保持顺序版本）
func (p *BatchTaskProcessor) ProcessTasks(ctx context.Context, tasks []Task) (*ProcessTasksResult, error) {
	startTime := time.Now()
	taskNum := len(tasks)

	if taskNum == 0 {
		return &ProcessTasksResult{
			Orders:   []Order{},
			Errors:   []error{},
			Duration: time.Since(startTime),
		}, nil
	}

	// 创建带缓冲的通道
	resultCh := make(chan OrderWithSeq, taskNum)
	defer close(resultCh)

	// 使用WaitGroup等待所有协程完成
	var wg sync.WaitGroup

	// 创建超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	// 启动协程执行任务
	for i, task := range tasks {
		wg.Add(1)
		go func(seq int, t Task) {
			defer func() {
				wg.Done()
				// 处理panic
				if r := recover(); r != nil {
					err := fmt.Errorf("task %d panic: %v", t.GetID(), r)
					resultCh <- OrderWithSeq{
						Seq:   seq,
						Order: Order{},
						Error: err,
					}
				}
			}()

			// 在协程内检查超时
			select {
			case <-timeoutCtx.Done():
				resultCh <- OrderWithSeq{
					Seq:   seq,
					Order: Order{},
					Error: fmt.Errorf("task %d timeout", t.GetID()),
				}
				return
			default:
			}

			// 执行任务
			order, err := t.Execute()
			resultCh <- OrderWithSeq{
				Seq:   seq,
				Order: order,
				Error: err,
			}
		}(i, task)
	}

	// 等待所有协程完成
	go func() {
		wg.Wait()
		// 这里不能close(resultCh)，因为已经defer close了
	}()

	// 收集结果
	results := make([]OrderWithSeq, 0, taskNum)
	timeout := time.NewTimer(p.Timeout)
	defer timeout.Stop()

	for i := 0; i < taskNum; i++ {
		select {
		case result := <-resultCh:
			results = append(results, result)
		case <-timeout.C:
			return nil, errors.New("batch processing timeout")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 如果需要保持顺序，则排序
	if p.KeepOrder {
		sort.Sort(BySeq(results))
	}

	// 分离成功和失败的结果
	var orders []Order
	var errors []error
	successCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
			errorCount++
		} else {
			orders = append(orders, result.Order)
			successCount++
		}
	}

	return &ProcessTasksResult{
		Orders:       orders,
		Errors:       errors,
		Duration:     time.Since(startTime),
		SuccessCount: successCount,
		ErrorCount:   errorCount,
	}, nil
}

// ProcessTasksSimple 简化版本（不保持顺序，快速失败）
func (p *BatchTaskProcessor) ProcessTasksSimple(ctx context.Context, tasks []Task) ([]Order, error) {
	taskNum := len(tasks)
	if taskNum == 0 {
		return []Order{}, nil
	}

	orderCh := make(chan Order, taskNum)
	errCh := make(chan error, taskNum)
	defer close(orderCh)
	defer close(errCh)

	var wg sync.WaitGroup

	// 启动协程执行任务
	for _, task := range tasks {
		wg.Add(1)
		go func(t Task) {
			defer func() {
				wg.Done()
				if r := recover(); r != nil {
					errCh <- fmt.Errorf("task %d panic: %v", t.GetID(), r)
				}
			}()

			order, err := t.Execute()
			if err != nil {
				errCh <- err
				return
			}
			orderCh <- order
		}(task)
	}

	// 等待所有协程完成
	go func() {
		wg.Wait()
	}()

	// 收集结果
	var orders []Order
	timeout := time.NewTimer(p.Timeout)
	defer timeout.Stop()

	for i := 0; i < taskNum; i++ {
		select {
		case order := <-orderCh:
			orders = append(orders, order)
		case err := <-errCh:
			return nil, err // 快速失败
		case <-timeout.C:
			return nil, errors.New("batch processing timeout")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return orders, nil
}

// 实际使用场景演示
func Test_main(t *testing.T) {
	// 创建测试任务
	tasks := []Task{
		&OrderTask{ID: 1, Name: "ProductA", Duration: 100 * time.Millisecond, ShouldError: false},
		&OrderTask{ID: 2, Name: "ProductB", Duration: 200 * time.Millisecond, ShouldError: false},
		&OrderTask{ID: 3, Name: "ProductC", Duration: 150 * time.Millisecond, ShouldError: true}, // 模拟错误
	}

	// 创建批量处理器
	processor := &BatchTaskProcessor{
		MaxConcurrency: 10,
		Timeout:        5 * time.Second,
		KeepOrder:      true, // 保持顺序
	}

	ctx := context.Background()

	fmt.Println("=== 场景1: 完整结果处理（容错，保持顺序） ===")
	result, err := processor.ProcessTasks(ctx, tasks)
	if err != nil {
		log.Printf("批量处理失败: %v", err)
		return
	}

	fmt.Printf("处理完成 - 成功: %d, 失败: %d, 耗时: %v\n",
		result.SuccessCount, result.ErrorCount, result.Duration)

	fmt.Println("\n成功的订单:")
	for _, order := range result.Orders {
		fmt.Printf("  Order ID: %d, Name: %s\n", order.ID, order.Name)
	}

	fmt.Println("\n失败的任务:")
	for _, err := range result.Errors {
		fmt.Printf("  Error: %v\n", err)
	}

	// 转换为JSON输出
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("\nJSON结果:\n%s\n", jsonResult)

	fmt.Println("\n=== 场景2: 简单处理（快速失败） ===")
	// 创建只有成功任务的列表用于演示快速处理
	successTasks := []Task{
		&OrderTask{ID: 1, Name: "ProductA", Duration: 100 * time.Millisecond},
		&OrderTask{ID: 2, Name: "ProductB", Duration: 200 * time.Millisecond},
		&OrderTask{ID: 3, Name: "ProductC", Duration: 150 * time.Millisecond},
	}

	orders, err := processor.ProcessTasksSimple(ctx, successTasks)
	if err != nil {
		log.Printf("简单处理失败: %v", err)
	} else {
		fmt.Printf("简单处理成功，获得 %d 个订单\n", len(orders))
		for _, order := range orders {
			fmt.Printf("  Order ID: %d, Name: %s\n", order.ID, order.Name)
		}
	}
}
