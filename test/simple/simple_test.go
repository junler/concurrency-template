package test

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

// Order 表示任务信息
type Order struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

// OrderWithSeq 用于保存带序号的结果以保持顺序
type OrderWithSeq struct {
	Seq       int
	OrderItem Order
}

// TaskResult 用于记录每个任务的结果（成功或失败）
type TaskResult struct {
	Seq       int
	Order     Order
	Error     error
	IsSuccess bool
}

// BySeq 实现 sort.Interface 用于按序号排序
type BySeq []TaskResult

func (a BySeq) Len() int           { return len(a) }
func (a BySeq) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySeq) Less(i, j int) bool { return a[i].Seq < a[j].Seq }

// processTask 模拟处理单个订单（例如 API 调用）
func processTask(task Order) (Order, error) {
	// 模拟耗时操作（例如 API 调用或数据库查询）
	time.Sleep(time.Millisecond * 500) // 模拟处理时间
	if task.Id%3 == 0 {                // 模拟每第三个任务失败
		return Order{}, fmt.Errorf("处理订单 %d 失败", task.Id)
	}
	// 模拟成功处理
	return Order{
		Name: fmt.Sprintf("已处理_%s", task.Name),
		Id:   task.Id,
	}, nil
}

// processBatchOrders 批量处理订单
func processBatchOrders(orders []Order) ([]TaskResult, error) {
	taskNum := len(orders)
	if taskNum == 0 {
		return nil, nil
	}

	// 初始化通道和 WaitGroup
	resultCh := make(chan TaskResult, taskNum) // 用于接收任务结果（成功或失败）
	var wg sync.WaitGroup

	// 设置每个任务的超时时间
	timeoutTime := time.Second * 3
	taskTimer := time.NewTimer(timeoutTime)
	defer taskTimer.Stop()

	// 为每个任务启动协程
	for i, order := range orders {
		wg.Add(1)
		go func(seq int, task Order) {
			defer func() {
				wg.Done()
				if r := recover(); r != nil {
					err := fmt.Errorf("系统 panic: %v", r)
					resultCh <- TaskResult{
						Seq:       seq,
						Order:     Order{},
						Error:     err,
						IsSuccess: false,
					}
				}
			}()

			// 执行任务
			result, err := processTask(task)
			resultCh <- TaskResult{
				Seq:       seq,
				Order:     result,
				Error:     err,
				IsSuccess: err == nil,
			}
		}(i, order)
	}

	// 等待所有协程完成并关闭通道
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集结果并处理超时
	var taskResults []TaskResult
	for i := 0; i < taskNum; i++ {
		select {
		case <-taskTimer.C:
			return nil, errors.New("任务超时")
		case result := <-resultCh:
			taskResults = append(taskResults, result)
		}
		taskTimer.Reset(timeoutTime)
	}

	// 按序号排序以保持结果顺序
	sort.Sort(BySeq(taskResults))

	return taskResults, nil
}

func Test_main(t *testing.T) {
	// 构造示例输入订单
	orderNum := 100
	orders := make([]Order, orderNum)
	for i := 0; i < orderNum; i++ {
		orders[i] = Order{
			Name: fmt.Sprintf("订单_%d", i+1),
			Id:   i + 1,
		}
	}

	// 执行批量订单处理
	taskResults, err := processBatchOrders(orders)
	if err != nil {
		fmt.Printf("批量处理错误: %v\n", err)
		return
	}

	// 统计成功和失败任务
	successCount := 0
	failureCount := 0
	var successOrders []Order
	var errorsList []error

	for _, result := range taskResults {
		if result.IsSuccess {
			successOrders = append(successOrders, result.Order)
			successCount++
		} else {
			if result.Error != nil {
				errorsList = append(errorsList, result.Error)
				failureCount++
			}
		}
	}

	// 在 main 中打印统计结果和详细信息
	fmt.Println("=== 任务处理结果 ===")
	fmt.Printf("总任务数: %d\n", len(orders))
	fmt.Printf("成功任务数: %d\n", successCount)
	fmt.Printf("失败任务数: %d\n", failureCount)

	fmt.Println("\n成功任务详情:")
	for _, result := range taskResults {
		if result.IsSuccess {
			fmt.Printf("任务 %d: 订单 %+v\n", result.Seq+1, result.Order)
		}
	}
	fmt.Println("\n失败任务详情:")
	for _, result := range taskResults {
		if !result.IsSuccess && result.Error != nil {
			fmt.Printf("任务 %d: 错误 %v\n", result.Seq+1, result.Error)
		}
	}
	fmt.Println("\n最终成功订单列表:")
	for _, order := range successOrders {
		fmt.Printf("订单: %+v\n", order)
	}
}
