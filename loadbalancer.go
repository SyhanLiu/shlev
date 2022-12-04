package shlev

import (
	"hash/crc32"
	"net"
)

// LoadBalancing 负载均衡算法，默认为轮询
type LoadBalancing int

const (
	// RoundRobin 轮询
	RoundRobin LoadBalancing = iota

	// LeastConnections 最小连接
	LeastConnections

	// SourceAddrHash hash值
	SourceAddrHash
)

// loadBalancer 负载均衡接口
type loadBalancer interface {
	register(*EventLoop)
	next(net.Addr) *EventLoop
	iterate(func(int, *EventLoop) bool)
	len() int
}

// roundRobinLoadBalancer 轮询负载均衡
type roundRobinLoadBalancer struct {
	nextIndex  int
	eventLoops []*EventLoop
	size       int
}

// leastConnectionsLoadBalancer 最少连接负载均衡
type leastConnectionsLoadBalancer struct {
	eventLoops []*EventLoop
	size       int
}

// sourceAddrHashLoadBalancer hash负载均衡
type sourceAddrHashLoadBalancer struct {
	eventLoops []*EventLoop
	size       int
}

// ==================================== 轮询负载均衡接口实现 ====================================
// 注册
func (lb *roundRobinLoadBalancer) register(e *EventLoop) {
	e.index = lb.size
	lb.eventLoops = append(lb.eventLoops, e)
	lb.size++
}

// next returns the eligible event-loop based on Round-Robin algorithm.
func (lb *roundRobinLoadBalancer) next(_ net.Addr) (e *EventLoop) {
	e = lb.eventLoops[lb.nextIndex]
	if lb.nextIndex++; lb.nextIndex >= lb.size {
		lb.nextIndex = 0
	}
	return
}

func (lb *roundRobinLoadBalancer) iterate(f func(int, *EventLoop) bool) {
	for i, el := range lb.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (lb *roundRobinLoadBalancer) len() int {
	return lb.size
}

// ================================= 最小连接负载均衡接口实现 =================================
func (lb *leastConnectionsLoadBalancer) min() (el *EventLoop) {
	el = lb.eventLoops[0]
	minN := el.loadConn()
	for _, v := range lb.eventLoops[1:] {
		if n := v.loadConn(); n < minN {
			minN = n
			el = v
		}
	}
	return
}

func (lb *leastConnectionsLoadBalancer) register(el *EventLoop) {
	el.index = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// next 返回可用的EventLoop
func (lb *leastConnectionsLoadBalancer) next(_ net.Addr) (el *EventLoop) {
	return lb.min()
}

func (lb *leastConnectionsLoadBalancer) iterate(f func(int, *EventLoop) bool) {
	for i, el := range lb.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (lb *leastConnectionsLoadBalancer) len() int {
	return lb.size
}

// ======================================= 哈希负载均衡接口实现 ========================================
func (lb *sourceAddrHashLoadBalancer) register(el *EventLoop) {
	el.index = lb.size
	lb.eventLoops = append(lb.eventLoops, el)
	lb.size++
}

// hash 算hash值
func (lb *sourceAddrHashLoadBalancer) hash(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	return -v
}

func (lb *sourceAddrHashLoadBalancer) next(netAddr net.Addr) *EventLoop {
	hashCode := lb.hash(netAddr.String())
	return lb.eventLoops[hashCode%lb.size]
}

func (lb *sourceAddrHashLoadBalancer) iterate(f func(int, *EventLoop) bool) {
	for i, el := range lb.eventLoops {
		if !f(i, el) {
			break
		}
	}
}

func (lb *sourceAddrHashLoadBalancer) len() int {
	return lb.size
}
