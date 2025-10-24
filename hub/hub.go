package hub

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pl "tunnels/protocol"

	"github.com/cloudwego/netpoll"
)

type connkey struct{}

var ctxkey = connkey{}

type Hub struct {
	lis netpoll.Listener

	network string
	address string

	timeout time.Duration

	sessionMar *SessionManager
}

func NewHub(opts ...HubOption) *Hub {
	srv := &Hub{
		network:    "tcp",
		address:    ":0",
		timeout:    1 * time.Second,
		sessionMar: NewManager(),
	}
	srv.init(opts...)

	return srv
}

func (s *Hub) init(opts ...HubOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Hub) Start(ctx context.Context) error {
	if err := s.listen(); err != nil {
		fmt.Println(err.Error())
		return err
	}
	slog.InfoContext(ctx, "tunnle server listening on: "+s.lis.Addr().String())
	// LogInfof("server listening on: %s", s.lis.Addr().String())

	// new server
	opts := []netpoll.Option{
		// netpoll.WithOnPrepare(prepare),
		// // netpoll.WithOnConnect(onConnect),
		// netpoll.WithOnDisconnect(handleDisconnect),
	}
	eventLoop, err := netpoll.NewEventLoop(s.handleConnection, opts...)
	if err != nil {
		panic(err)
	}
	// start listen loop ...
	return eventLoop.Serve(s.lis)
}

func (s *Hub) listen() error {
	if s.lis == nil {
		lis, err := netpoll.CreateListener(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	return nil
}

// 初始化连接
func prepare(conn netpoll.Connection) (ctx context.Context) {
	mc := newMuxConn(conn)
	ctx = context.WithValue(context.Background(), ctxkey, mc)
	return ctx
}
func (h *Hub) handleConnection(ctx context.Context, conn netpoll.Connection) (err error) {
	// mc := ctx.Value(ctxkey).(*muxConn)
	// defer mc.clear()
	defer conn.Close()
	reader := conn.Reader()

	var initPacket pl.InitPacket
	slog.Info("Processing initialization packet", "length", reader.Len())
	if err = pl.ProcessRequest(reader, &initPacket); err != nil {
		slog.Error("Initialization connection failed", "err", err)
		return
	}

	session := h.sessionMar.GetOrCreate(initPacket.DeviceID)
	// conn.Writer()
	fmt.Printf("%+v\n", initPacket)
	switch initPacket.Role {
	case pl.ROLE_SENDER:

		session.SetSender(conn)
		// writer := conn.Writer()
		// // go func() {
		// // writer := netpoll.NewLinkBuffer()
		// for i := 0; i < 5; i++ {
		// 	writer.WriteBinary([]byte("111111"))
		// 	writer.Flush()
		// 	// writer.MallocAck(writer.MallocLen())
		// 	// time.Sleep(time.Millisecond * 1)
		// 	// mc.wqueue.Add(func() (buf netpoll.Writer, isNil bool) {
		// 	// 	return writer, false
		// 	// })
		// }
		// }()
		// for i := 0; i < 5; i++ {
		// 	writer.WriteString("hello sender1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111")
		// 	mc.wqueue.Add(func() (buf netpoll.Writer, isNil bool) {
		// 		return writer, false
		// 	})
		// 	writer.Flush()
		// time.Sleep(time.Millisecond * 5)
		// time.Sleep(time.Second * 2)

		// writer.WriteString("11111")
		// // mc.Put(writer)
		// writer.Flush()

		// writer.WriteString("333")
		// // mc.Put(writer)
		// writer.Flush()
		slog.Info("[Sender] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	case pl.ROLE_RECEIVER:
		// writer.WriteString("222")
		// mc.Put(writer)
		slog.Info("[Receiver] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	default:
		return
	}

	time.Sleep(time.Second * 20)
	// for i := range 5 {
	// 	fmt.Println(i)
	// 	// handle connection
	// }
	return
}

func handleDisconnect(ctx context.Context, conn netpoll.Connection) {
	mc := ctx.Value(ctxkey).(*muxConn)
	slog.Info("关闭连接", "role", mc.Role)
}
