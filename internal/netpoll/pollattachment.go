package netpoll

import "sync"

var pollAttachmentPool = sync.Pool{
	New: func() interface{} {
		return &PollAttachment{
			FD:       0,
			Callback: nil,
		}
	},
}

// GetPollAttachment 从缓存池中获取PollAttachment
func GetPollAttachment() *PollAttachment {
	return pollAttachmentPool.Get().(*PollAttachment)
}

// PutPollAttachment 放入缓存池
func PutPollAttachment(pa *PollAttachment) {
	if pa == nil {
		return
	}
	pa.FD, pa.Callback = 0, nil
	pollAttachmentPool.Put(pa)
}

type PollEventHandler func(int, uint32) error

// PollAttachment is the user data which is about to be stored in "void *ptr" of epoll_data or "void *udata" of kevent.
type PollAttachment struct {
	FD       int
	Callback PollEventHandler
}
