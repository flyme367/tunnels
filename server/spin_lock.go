package server

import (
	"runtime"
	"sync/atomic"
)

// SpinLock 自旋锁
type spinLock struct {
	state int32
}

// Lock 获取锁
func (s *spinLock) Lock() {
	for !atomic.CompareAndSwapInt32(&s.state, 0, 1) {
		// 在等待锁的时候，让出当前的时间片，避免占用过多CPU
		runtime.Gosched()
	}
}

// Unlock 释放锁
func (s *spinLock) Unlock() {
	atomic.StoreInt32(&s.state, 0)
}
