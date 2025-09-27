package server

import (
	"sync"
	"time"
)

var tickerPool = sync.Pool{
	New: func() any {
		// 这里的时间间隔不重要，因为后面会重置
		return time.NewTicker(time.Second)
	},
}

// GetTicker 从池中获取一个Ticker，并重置为新的时间间隔
func GetTicker(d time.Duration) *time.Ticker {
	t := tickerPool.Get().(*time.Ticker)
	t.Reset(d)
	return t
}

// ReleaseTicker 将Ticker放回池中
func ReleaseTicker(t *time.Ticker) {
	t.Stop()
	// 清空channel中的事件
	for {
		select {
		case <-t.C:
		default:
			tickerPool.Put(t)
			return
		}
	}
}
