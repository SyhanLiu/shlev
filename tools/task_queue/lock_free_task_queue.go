package task_queue

import (
	"sync/atomic"
	"unsafe"
)

// TODO 无锁队列学习博客：https://coolshell.cn/articles/8239.html

// lockFreeTaskQueue 无锁队列，参看gnet实现
type lockFreeTaskQueue struct {
	head   *node
	tail   *node
	length int32
}

type node struct {
	value *Task
	next  *node
}

func NewLockFreeTaskQueue() AsyncTaskQueue {
	n := &node{}
	return &lockFreeTaskQueue{
		head:   n,
		tail:   n,
		length: 0,
	}
}

func (q *lockFreeTaskQueue) Enqueue(task *Task) {
	n := &node{value: task}
	for true {
		tail := loadNode(&q.tail)
		next := loadNode(&tail.next)

		// 尾部未移动
		if tail == loadNode(&q.tail) {
			if next == nil {
				if cas(&tail.next, next, n) { // 入队成功
					cas(&q.tail, tail, n)
					atomic.AddInt32(&q.length, 1)
					return
				}
			} else {
				// next不为空，意味着有其他线程把一个新task加入了队列，此时如果
				// 此时队列尾部没有指向最后一个节点，尝试把next存入队列尾部
				cas(&q.tail, tail, next)
			}
		} else {
			// 如果tail != &q.tail，说明指针已经移动，需要重新开始
			continue
		}
	}
}

func (q *lockFreeTaskQueue) Dequeue() *Task {
	for true {
		head := loadNode(&q.head)
		tail := loadNode(&q.tail)
		next := loadNode(&head.next)

		// 队头是否改变
		if head == loadNode(&q.head) {
			if head == tail {
				// 空队列
				if next == nil {
					return nil
				}
				cas(&q.tail, tail, next)
			} else {
				task := next.value
				if cas(&q.head, head, next) {
					// 出队成功
					atomic.AddInt32(&q.length, -1)
					return task
				}
			}
		} else {
			// 对头改变是重新操作
			continue
		}
	}
	return nil
}

// IsEmpty 判断队列是否为空
func (q *lockFreeTaskQueue) IsEmpty() bool {
	return atomic.LoadInt32(&(q.length)) == 0
}

// 加载节点
func loadNode(n **node) *node {
	return (*node)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(n))))
}

// cas操作
func cas(p **node, old, new *node) bool {
	return atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(p)), unsafe.Pointer(old), unsafe.Pointer(new))
}
