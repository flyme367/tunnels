package hub

import (
	"sync"
	"time"
)

var tickerPool = sync.Pool{
	New: func() any {
		return time.NewTicker(time.Second)
	},
}

// GetTicker 从池中获取一个Ticker，并重置为新的时间间隔
func GetTicker(d time.Duration) *time.Ticker {
	t := tickerPool.Get().(*time.Ticker)
	// 重置Ticker周期
	if t.C == nil {
		// 如果通道已关闭，创建新的Ticker
		t = time.NewTicker(d)
	} else {
		// 重置现有Ticker
		t.Reset(d)
	}
	return t
}

// ReleaseTicker 将Ticker放回池中
func ReleaseTicker(t *time.Ticker) {
	t.Stop()
	// 清空通道
	select {
	case <-t.C:
	default:
	}
	tickerPool.Put(t)
}

// var connPool = sync.Pool{
// 	New: func() any {
// 		return &muxConn{}
// 	},
// }
