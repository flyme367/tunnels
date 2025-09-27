package server

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/cloudwego/netpoll"
)

type connkey struct{}

var ctxkey connkey

type Server struct {
	lis netpoll.Listener

	network string
	address string

	timeout time.Duration

	sessionMar *SessionManager
	// rdb      *redis.Client
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:    "tcp",
		address:    ":0",
		timeout:    1 * time.Second,
		sessionMar: NewManager(),
	}
	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.listen(); err != nil {
		return err
	}
	slog.InfoContext(ctx, "tunnle server listening on: "+s.lis.Addr().String())
	// LogInfof("server listening on: %s", s.lis.Addr().String())

	// new server
	opts := []netpoll.Option{
		netpoll.WithOnPrepare(s.prepare),
		// netpoll.WithOnDisconnect(s.handleDisconnect),
	}
	eventLoop, err := netpoll.NewEventLoop(s.handleConnection, opts...)
	if err != nil {
		panic(err)
	}
	// start listen loop ...
	return eventLoop.Serve(s.lis)
}

func (s *Server) Endpoint() (*url.URL, error) {
	return nil, nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.InfoContext(ctx, "tunnle server stopping")
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}

	return nil
}

func (s *Server) listen() error {
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
func (s *Server) prepare(conn netpoll.Connection) (ctx context.Context) {
	reader := conn.Reader()
	req := &InitPacket{}
	if err := ProcessRequest(reader, req); err != nil {
		slog.Error("Initialization connection failed", "err", err)
		conn.Close()
		return
	}

	if req.Header.Cmd != CMD_INIT {
		slog.Error("Invalid init packet: unexpected command", "cmd", req.Header.CRC)
		conn.Close()
		return
	}

	//创建会话空间
	session := s.sessionMar.GetOrCreate(req.DeviceID)
	switch req.Role {
	case ROLE_SENDER:
		session.SetSender(conn)
		slog.InfoContext(ctx, "Sender connected for device", "deviceID", req.DeviceID)
	case ROLE_RECEIVER:
		session.SetSender(conn)
		slog.InfoContext(ctx, "Receiver connected for device", "deviceID", req.DeviceID)
	}
	// fmt.Println(111111111111111111)
	// mc := newMuxConn(conn)
	ctx = context.Background()
	// ctx := context.WithValue(context.Background(), ctxkey, mc)
	// time.Sleep(time.Second * 10)
	return
}

func (s *Server) handleConnection(ctx context.Context, conn netpoll.Connection) (err error) {
	return
}

func (s *Server) handleDisconnect(ctx context.Context, conn netpoll.Connection) {

}
