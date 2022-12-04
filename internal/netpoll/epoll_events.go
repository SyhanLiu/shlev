package netpoll

import "golang.org/x/sys/unix"

/*
** 水平触发(level-triggered):
** socket接收缓冲区不为空 有数据可读 读事件一直触发
** socket发送缓冲区不满 可以继续写入数据 写事件一直触发
**
** 边沿触发(edge-triggered):
** socket的接收缓冲区状态变化时触发读事件，即空的接收缓冲区刚接收到数据时触发读事件
** socket的发送缓冲区状态变化时触发写事件，即满的缓冲区刚空出空间时触发读事件
** 边沿触发仅触发一次，水平触发会一直触发。
 */

const (
	readEvents      = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents     = unix.EPOLLOUT
	readWriteEvents = readEvents | writeEvents

	// ErrEvents 表示非读写的套接字异常事件
	// EPOLLERR 和 EPOLLHUP会自动监听，无需手动设置，但是我就是要写！！
	// unix.EPOLLERR：向已经关闭的socket写或者读
	// unix.EPOLLHUP：对端关闭了套接字
	/*
	** unix.EPOLLRDHUP：在对端关闭时会触发，或者对端shutdown写
	** 对EPOLLRDHUP的处理应该放在EPOLLIN和EPOLLOUT前面，处理方式应该 是close掉相应的fd后，作其他应用层的清理动作；
	** 如果采用的是LT触发模式，且没有close相应的fd, EPOLLRDHUP会持续被触发；
	** EPOLLRDHUP想要被触发，需要显式地在epoll_ctl调用时设置在events中
	 */
	ErrEvents = unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP

	/*
	** unix.EPOLLOUT
	** 有写需要时才通过epoll_ctl添加相应fd，不然在LT模式下会频繁触发;
	** 对于写操作，大部分情况下都处于可写状态，可先直接调用write来发送数据，直到返回 EAGAIN后再使能EPOLLOUT，待触发后再继续write。
	 */
	// OutEvents combines EPOLLOUT event and some exceptional events.
	OutEvents = ErrEvents | unix.EPOLLOUT
	// InEvents combines EPOLLIN/EPOLLPRI events and some exceptional events.
	InEvents = ErrEvents | unix.EPOLLIN | unix.EPOLLPRI
)

const (
	// MaxTasksOnce 每次处理的普通任务数量
	MaxTasksOnce = 100
)

type EventsList struct {
	events []unix.EpollEvent
}

func newEventsList() *EventsList {
	return &EventsList{
		events: make([]unix.EpollEvent, 1024),
	}
}
