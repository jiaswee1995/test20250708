package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// 任务函数定义：接收 context，可被取消
type Task func(ctx context.Context) error

// 日志信息结构
type TaskLog struct {
	ID       int
	Start    time.Time
	End      time.Time
	Duration time.Duration
	Success  bool
	Error    error
}

// 调度器函数
func RunTasks(tasks []Task, maxConcurrency int, taskTimeout, overallTimeout time.Duration) []TaskLog {
	var (
		logs              = make([]TaskLog, len(tasks))
		wg                sync.WaitGroup
		sem               = make(chan struct{}, maxConcurrency) // 控制并发
		ctxAll, cancelAll = context.WithTimeout(context.Background(), overallTimeout)
	)
	defer cancelAll()

	for i, task := range tasks {
		i, task := i, task
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}: // acquire
				defer func() { <-sem }() // release

				logEntry := TaskLog{ID: i, Start: time.Now()}
				ctxTask, cancelTask := context.WithTimeout(ctxAll, taskTimeout)
				defer cancelTask()

				err := task(ctxTask)
				logEntry.End = time.Now()
				logEntry.Duration = logEntry.End.Sub(logEntry.Start)
				logEntry.Success = err == nil
				logEntry.Error = err

				logs[i] = logEntry

			case <-ctxAll.Done(): // 整体超时提前结束
				logs[i] = TaskLog{
					ID:      i,
					Start:   time.Now(),
					End:     time.Now(),
					Success: false,
					Error:   fmt.Errorf("scheduler timeout before start"),
				}
			}
		}()
	}

	wg.Wait()
	return logs
}

// 模拟任务：随机执行 1~3 秒，可被取消
func mockTask(id int) Task {
	return func(ctx context.Context) error {
		sleep := time.Duration(rand.Intn(3)+1) * time.Second
		select {
		case <-ctx.Done():
			return fmt.Errorf("task %d timeout/cancelled", id)
		case <-time.After(sleep):
			return nil
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// 创建任务
	numTasks := 10
	tasks := make([]Task, numTasks)
	for i := 0; i < numTasks; i++ {
		tasks[i] = mockTask(i)
	}

	// 参数设置
	maxConcurrency := 3 // 最多同时执行 3 个
	taskTimeout := 2 * time.Second
	overallTimeout := 2 * time.Second

	start := time.Now()
	results := RunTasks(tasks, maxConcurrency, taskTimeout, overallTimeout)
	end := time.Now()

	fmt.Printf("All tasks returned in: %v\n", end.Sub(start))

	// 打印日志
	for _, logEntry := range results {
		status := "SUCCESS"
		if !logEntry.Success {
			status = "FAILED"
		}
		log.Printf("[Task %d] Start: %s | End: %s | Duration: %v | Status: %s | Error: %v",
			logEntry.ID, logEntry.Start.Format(time.RFC3339), logEntry.End.Format(time.RFC3339),
			logEntry.Duration, status, logEntry.Error)
	}
}
