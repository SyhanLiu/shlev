package netpoll

import "shlev/tools/task_queue"

type Netpoller interface {
	// Delete 删除套接字
	Delete(fd int) error
	// AddRead 添加读
	AddRead(fd int) error
	// AddWrite 添加写
	AddWrite(fd int) error
	// ModRead 改为读
	ModRead(fd int) error
	// Polling 轮询事件
	Polling(func(int, uint32) error) error
	// Close 关闭事件循环
	Close() error
	// Open 开启poll
	Open() error
	// AddUrgentTask 添加紧急任务
	AddUrgentTask(task_queue.TaskFunc, interface{}) error
	// AddTask 添加普通任务
	AddTask(task_queue.TaskFunc, interface{}) error
}
