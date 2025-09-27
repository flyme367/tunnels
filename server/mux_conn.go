package server

import (
	"time"

	"github.com/cloudwego/netpoll"
	"github.com/cloudwego/netpoll/mux"
)

// type muxServer struct{}

type muxConn struct {
	netpoll.Connection
	wqueue *mux.ShardQueue // use for write
	Ts     time.Time
}

// var ShardSize int

// func init() {
// 	ShardSize = runtime.GOMAXPROCS(0)
// }

// Put puts the buffer getter back to the queue.
func (c *muxConn) Put(gt mux.WriterGetter) {
	c.wqueue.Add(gt)
}
func (c *muxConn) clear() {
	c.Close()
	c.wqueue.Close()
}

func newMuxConn(conn netpoll.Connection) *muxConn {
	mc := &muxConn{
		Connection: conn,
		wqueue:     mux.NewShardQueue(mux.ShardSize, conn),
		Ts:         time.Now(),
	}
	return mc
}
