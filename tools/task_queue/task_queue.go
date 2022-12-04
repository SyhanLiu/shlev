package task_queue

import "sync"

// TaskFunc 任务回调函数
type TaskFunc func(interface{}) error

// Task 包含回调函数和入参，Run(Arg)
type Task struct {
	Run TaskFunc
	Arg interface{}
}

var taskPool = sync.Pool{New: func() interface{} { return new(Task) }}

// GetTask 封装Get操作
func GetTask() *Task {
	return taskPool.Get().(*Task)
}

// PutTask 封装Put操作
func PutTask(task *Task) {
	task.Run, task.Arg = nil, nil
	taskPool.Put(task)
}

// AsyncTaskQueue 任务队列
type AsyncTaskQueue interface {
	Enqueue(*Task)
	Dequeue() *Task
	IsEmpty() bool
}
