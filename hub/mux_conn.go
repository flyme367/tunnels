package hub

import (
	"github.com/cloudwego/netpoll"
	"github.com/cloudwego/netpoll/mux"
)

// type muxServer struct{}

type muxConn struct {
	Role uint8
	netpoll.Connection
	wqueue *mux.ShardQueue // use for write
	// Ts     time.Time
}

// Put puts the buffer getter back to the queue.
func (c *muxConn) Put(writer netpoll.Writer) {
	c.wqueue.Add(func() (buf netpoll.Writer, isNil bool) {
		return writer, false
	})
}
func (c *muxConn) clear() {
	c.Close()
	c.wqueue.Close()
}

func newMuxConn(conn netpoll.Connection) *muxConn {
	mc := &muxConn{
		Connection: conn,
		wqueue:     mux.NewShardQueue(mux.ShardSize, conn),
		// Ts:         time.Now(),
	}
	return mc
}
