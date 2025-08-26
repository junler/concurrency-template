package models

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Order 订单模型
type Order struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	CustomerID  string     `json:"customer_id" gorm:"size:100;not null"`
	ProductName string     `json:"product_name" gorm:"size:200;not null"`
	Quantity    int        `json:"quantity" gorm:"not null"`
	Price       float64    `json:"price" gorm:"type:decimal(10,2);not null"`
	Status      string     `json:"status" gorm:"size:50;default:'pending'"`
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// APICall API调用记录
type APICall struct {
	ID           uint      `json:"id" gorm:"primarykey"`
	URL          string    `json:"url" gorm:"size:500;not null"`
	Method       string    `json:"method" gorm:"size:10;not null"`
	Status       string    `json:"status" gorm:"size:50;default:'pending'"`
	ResponseCode int       `json:"response_code"`
	ResponseBody string    `json:"response_body" gorm:"type:text"`
	Duration     int64     `json:"duration"` // 毫秒
	Error        string    `json:"error" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FileTask 文件处理任务
type FileTask struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	FileName    string     `json:"file_name" gorm:"size:255;not null"`
	FilePath    string     `json:"file_path" gorm:"size:500;not null"`
	FileSize    int64      `json:"file_size"`
	Status      string     `json:"status" gorm:"size:50;default:'pending'"`
	ProcessType string     `json:"process_type" gorm:"size:50;not null"` // compress, resize, convert等
	Result      string     `json:"result" gorm:"type:text"`
	Error       string     `json:"error" gorm:"type:text"`
	ProcessedAt *time.Time `json:"processed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// BatchJobResult 批量任务结果
type BatchJobResult struct {
	ID           uint       `json:"id" gorm:"primarykey"`
	JobType      string     `json:"job_type" gorm:"size:50;not null"` // order, api, file
	TotalTasks   int        `json:"total_tasks"`
	SuccessTasks int        `json:"success_tasks"`
	FailedTasks  int        `json:"failed_tasks"`
	Duration     int64      `json:"duration"` // 毫秒
	Status       string     `json:"status" gorm:"size:50;default:'running'"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// InitDB 初始化数据库
func InitDB() (*gorm.DB, error) {
	// 使用SQLite数据库
	db, err := gorm.Open(sqlite.Open("concurrency_app.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 自动迁移模式
	err = db.AutoMigrate(&Order{}, &APICall{}, &FileTask{}, &BatchJobResult{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
