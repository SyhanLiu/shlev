package task_queue_test

import (
	taskqueue2 "shlev/tools/task_queue"
	"sync"
	"sync/atomic"
	"testing"
)

func TestLockFreeTaskQueue(t *testing.T) {
	q := taskqueue2.NewLockFreeTaskQueue()
	wg := sync.WaitGroup{}
	wg.Add(4)
	var f int32
	go func() {
		for i := 0; i < 10000; i++ {
			task := &taskqueue2.Task{}
			q.Enqueue(task)
		}
		atomic.AddInt32(&f, 1)
		wg.Done()
	}()
	go func() {
		for i := 0; i < 10000; i++ {
			task := &taskqueue2.Task{}
			q.Enqueue(task)
		}
		atomic.AddInt32(&f, 1)
		wg.Done()
	}()

	var counter int32
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				atomic.AddInt32(&counter, 1)
			}
			if task == nil && atomic.LoadInt32(&f) == 2 {
				break
			}
		}
		wg.Done()
	}()
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				atomic.AddInt32(&counter, 1)
			}
			if task == nil && atomic.LoadInt32(&f) == 2 {
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()
	t.Logf("sent and received all %d tasks", counter)
}
