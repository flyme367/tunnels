package hub

import (
	"runtime"
	"sync/atomic"
)

// SpinLock 自旋锁
type spinLock struct {
	state      uint32
	backoff    uint8
	maxBackoff uint8 // 增加退避时间
}

func newSpinLock() *spinLock {
	return &spinLock{
		backoff:    1,
		maxBackoff: 16,
	}
}

// Lock 获取锁
func (s *spinLock) Lock() {

	// 自旋等待
	for {
		// 尝试获取锁
		if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
			return
		}

		// 指数退避策略
		for range s.backoff {
			runtime.Gosched() // 让出CPU时间片
		}

		// 增加退避时间（指数增长，上限maxBackoff）
		if s.backoff < s.maxBackoff {
			s.backoff <<= 1
		}
	}
}

// Unlock 释放锁
func (s *spinLock) Unlock() {
	atomic.StoreUint32(&s.state, 0)
}
