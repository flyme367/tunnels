package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/cloudwego/netpoll"
)

type connkey struct{}

var ctxkey = connkey{}

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
		fmt.Println(err.Error())
		return err
	}
	slog.InfoContext(ctx, "tunnle server listening on: "+s.lis.Addr().String())
	// LogInfof("server listening on: %s", s.lis.Addr().String())

	// new server
	opts := []netpoll.Option{
		netpoll.WithOnPrepare(prepare),
		// netpoll.WithOnConnect(onConnect),
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

// func onConnect(ctx context.Context, upstream netpoll.Connection) context.Context {
// 	fmt.Printf("监听连接 %d\n", upstream.Reader().Len())
// 	return ctx
// }

// 初始化连接
func prepare(conn netpoll.Connection) (ctx context.Context) {
	mc := newMuxConn(conn)
	ctx = context.WithValue(context.Background(), ctxkey, mc)
	return ctx
}

func (s *Server) handleConnection(ctx context.Context, conn netpoll.Connection) (err error) {
	// defer conn.Close()
	mc := ctx.Value(ctxkey).(*muxConn)
	defer mc.clear()

	initPacket := InitPacket{}
	reader := conn.Reader()
	slog.Info("Processing initialization packet", "length", reader.Len())
	if err = ProcessRequest(reader, &initPacket); err != nil {
		slog.Error("Initialization connection failed", "err", err)
		return
	}

	// fmt.Printf("初始化包：%+v\n", req)
	seq := uint16(0)
	// 获取或创建session
	session := s.sessionMar.GetOrCreate(initPacket.DeviceID)

	// 根据角色设置连接
	var isReady bool
	switch initPacket.Role {
	case ROLE_SENDER:
		isReady = session.SetSender(mc)
		slog.Info("[Sender] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	case ROLE_RECEIVER:
		isReady = session.SetReceiver(mc)
		slog.Info("[Receiver] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	default:
		slog.Error("Invalid init packet: unexpected role", "role", initPacket.Role)
		return
	}

	// 发送连接成功状态
	writer := conn.Writer()
	response := &DatadPacket{
		Header: &Header{
			Cmd:   CMD_STATUS,
			Order: seq,
		},
		Data: []byte{byte(STATUS_CONNECTED)},
	}

	Encodex(writer, response)
	mc.Put(writer)
	// mc.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })
	seq++
	// 等待配对完成
	if !isReady {
		slog.Debug("Waiting for peer connection .....", "deviceID", initPacket.DeviceID)
		<-session.WaitReady()
		slog.Debug("Peer connection established .....", "deviceID", initPacket.DeviceID)
	}

	// response := &DatadPacket{
	// 	Header: &Header{
	// 		Cmd:   CMD_STATUS,
	// 		Order: 0,
	// 	},
	// }
	// 发送传输就绪状态
	response.Data = []byte{byte(STATUS_READY)}
	response.Header.Order = seq
	Encodex(writer, response)
	// mc.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })
	seq++
	// 根据角色处理连接
	// var forwardPacket DatadPacket
	switch initPacket.Role {
	case ROLE_SENDER:
		// 处理sender连接
		for {
			response, err1 := ForwardRequest(reader)
			if err1 != nil {
				// slog.Info("[sender] reader  Invalid packet ", "error", err1)
				s.sessionMar.Remove(initPacket.DeviceID)
				return
			}

			if response.Header.Cmd == CMD_DATA {
				// slog.Info("[Received] data from sender ", "IP", conn.RemoteAddr(), "data", string(response.Data))

				// 转发数据给receiver
				if session.Receiver != nil {
					// slog.Info("[Received]  Forwarding data to receiver from sender ", "deviceID", initPacket.DeviceID)
					// Use receiver's sequence number for forwarded packets
					slog.Info("[Sender]  Received  ", "Active", session.Receiver.IsActive())
					receiverSeq := session.GetReceiverSeq()
					response.Header.Order = uint16(receiverSeq)
					Encodex(session.Receiver.Writer(), response)
					session.IncrementReceiverSeq()
				}
				// slog.Info("[Sender]  Received  ", "Active", session.Receiver.IsActive())
			}
		}
	case ROLE_RECEIVER:
		for {
			response, err1 := ForwardRequest(reader)
			if err1 != nil {
				s.sessionMar.Remove(initPacket.DeviceID)
				return
			}

			if response.Header.Cmd == CMD_DATA {
				// 转发数据给receiver
				if session.Sender != nil {
					// slog.Info("[Sender]  Forwarding data to send for device ", "deviceID", initPacket.DeviceID)
					// Use receiver's sequence number for forwarded packets
					slog.Info("[Sender]  Received  ", "Active", session.Sender.IsActive())
					response.Header.Order = seq
					Encodex(session.Sender.Writer(), response)
				}
				seq++
			}
		}
	}
	return
}

// sendResponse sends a response packet and flushes the writer
func (s *Server) sendResponse(writer netpoll.Writer, packet *DatadPacket) error {
	Encodex(writer, packet)
	return writer.Flush()
}

// func (s *Server) handleDisconnect(ctx context.Context, conn netpoll.Connection) {

// }
