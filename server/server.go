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
		// netpoll.WithOnPrepare(prepare),
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
func onConnect(ctx context.Context, upstream netpoll.Connection) context.Context {
	fmt.Printf("监听连接 %d\n", upstream.Reader().Len())
	return ctx
}

// 初始化连接
func prepare(conn netpoll.Connection) (ctx context.Context) {
	// fmt.Println(conn.Reader().Len())
	// fmt.Println("___________________________________")

	mc := newMuxConn(conn)
	ctx = context.WithValue(context.Background(), ctxkey, mc)
	return ctx
	// buf := make([]byte, 1024)
	// 读取初始化包
	// var reader netpoll.Reader
	// for {
	// 	if reader = conn.Reader(); reader.Len() > 0 {
	// 		fmt.Println(11111)
	// 		break
	// 	}
	// 	time.Sleep(time.Millisecond * 1)
	// }
	// fmt.Println(22222)
	// // n, err := conn.Read(buf)
	// // if err != nil {
	// // 	log.Printf("Failed to read init packet: %v", err)
	// // 	return
	// // }
	// // // fmt.Printf("连接数据：%X\n", buf[:n])
	// // reader := conn.Reader()
	// req := &InitPacket{}
	// fmt.Printf("初始化包：%d\n", reader.Len())
	// if err := ProcessRequest(reader, req); err != nil {
	// 	slog.Error("Initialization connection failed", "err", err)
	// 	// conn.Close()
	// 	return
	// }

	// if req.Header.Cmd != CMD_INIT {
	// 	slog.Error("Invalid init packet: unexpected command", "cmd", req.Header.Cmd)
	// 	conn.Close()
	// 	return
	// }

	// //创建会话空间
	// session := s.sessionMar.GetOrCreate(req.DeviceID)

	// var (
	// 	muxConn *muxConn
	// 	role    int8
	// 	isReady bool
	// )
	// switch req.Role {
	// case ROLE_SENDER:
	// 	role = ROLE_SENDER
	// 	muxConn = newMuxConn(conn)
	// 	session.SetSender(muxConn)
	// 	slog.InfoContext(context.Background(), "Sender connected for device", "deviceID", req.DeviceID)
	// case ROLE_RECEIVER:
	// 	role = ROLE_RECEIVER
	// 	muxConn = newMuxConn(conn)
	// 	session.SetReceiver(muxConn)
	// 	slog.InfoContext(context.Background(), "Receiver connected for device", "deviceID", req.DeviceID)
	// default:
	// 	slog.Error("Invalid init packet: unexpected role", "role", req.Role)
	// 	conn.Close()
	// 	return
	// }
	// seq := uint16(0)
	// // 发送连接成功状态
	// writer := conn.Writer()
	// state := &StatusPacket1{
	// 	Status:  STATUS_CONNECTED,
	// 	CmdCode: CMD_STATUS,
	// 	Order:   seq,
	// }
	// Encodex(writer, state)

	// // 发送连接成功状态
	// muxConn.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })
	// seq++

	// // 等待配对完成
	// if !isReady {
	// 	fmt.Printf("Waiting for peer connection for device %s...\n", req.DeviceID)
	// 	<-session.WaitReady()
	// 	fmt.Printf("Peer connection established for device %s\n", req.DeviceID)
	// }

	// // 发送传输就绪状态
	// state.Status = STATUS_READY
	// state.Order = seq
	// Encodex(writer, state)
	// muxConn.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })
	// seq++

	// ctx = context.WithValue(context.Background(), ctxkey, role)

	// return context.Background()
}

func (s *Server) handleConnection(ctx context.Context, conn netpoll.Connection) (err error) {
	defer conn.Close()
	// mc := ctx.Value(ctxkey).(*muxConn)
	// defer mc.clear()

	initPacket := InitPacket{}
	reader := conn.Reader()
	fmt.Printf("初始化包长度：%d\n", reader.Len())
	if err = ProcessRequest(reader, &initPacket); err != nil {
		slog.Error("Initialization connection failed", "err", err)
		// conn.Close()
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
		isReady = session.SetSender(conn)
		slog.Debug("[Sender] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	case ROLE_RECEIVER:
		isReady = session.SetReceiver(conn)
		slog.Debug("[Receiver] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	default:
		slog.Error("Invalid init packet: unexpected role", "role", initPacket.Role)
		return
	}

	// 发送连接成功状态
	writer := conn.Writer()
	req := &DatadPacket{
		Header: &Header{
			Cmd:   CMD_STATUS,
			Order: seq,
		},
		Data: []byte{byte(STATUS_CONNECTED)},
	}

	Encodex(writer, req)
	// mc.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })
	seq++
	fmt.Println("dsadsadasdsasdsadasd_____________________")
	// 等待配对完成
	if !isReady {
		slog.Debug("Waiting for peer connection .....", "deviceID", initPacket.DeviceID)
		<-session.WaitReady()
		slog.Debug("Peer connection established .....", "deviceID", initPacket.DeviceID)
	}

	// 发送传输就绪状态
	req.Data = []byte{byte(STATUS_READY)}
	req.Header.Order = seq
	Encodex(writer, req)
	// mc.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })
	seq++
	fmt.Println("aaaaaaaaaaaaaaaaaaaaaa_____________________")
	// 根据角色处理连接
	var forwardPacket DatadPacket
	switch initPacket.Role {
	case ROLE_SENDER:
		// 处理sender连接
		for {
			err1 := ForwardRequest(reader, &forwardPacket)
			if err1 != nil {
				slog.DebugContext(ctx, "[sender] reader  Invalid packet ", "error", err1)
				continue
			}

			if forwardPacket.Header.Cmd == CMD_DATA {
				slog.Info("[Received] data from sender ", "IP", conn.RemoteAddr(), "data", string(forwardPacket.Data))

				// 转发数据给receiver
				if session.Receiver != nil {
					slog.Debug("[Received]  Forwarding data to receiver for device ", "deviceID", initPacket.DeviceID)
					// Use receiver's sequence number for forwarded packets
					receiverSeq := session.GetReceiverSeq()
					req.Header.Order = uint16(receiverSeq)
					req.Header.Cmd = forwardPacket.Header.Cmd
					req.Data = forwardPacket.Data
					// status.CmdCode = forwardPacket.Header.Cmd
					// status.Order = uint16(receiverSeq)
					// status.CmdCode
					Encodex(writer, req)
					session.IncrementReceiverSeq()
				}
			}
		}
	case ROLE_RECEIVER:
		for {
			err1 := ForwardRequest(reader, &forwardPacket)
			if err1 != nil {
				slog.DebugContext(ctx, "[receiver] reader  Invalid packet ", "error", err1)
				continue
			}

			if forwardPacket.Header.Cmd == CMD_DATA {
				// 转发数据给receiver
				if session.Sender != nil {
					slog.Debug("[Sender]  Forwarding data to receiver for device ", "deviceID", initPacket.DeviceID)
					// Use receiver's sequence number for forwarded packets

					req.Header.Order = seq
					req.Header.Cmd = forwardPacket.Header.Cmd
					req.Data = forwardPacket.Data
					// status.CmdCode = forwardPacket.Header.Cmd
					// status.Order = uint16(receiverSeq)
					// status.CmdCode
					Encodex(writer, req)
				}
				seq++
			}
		}
	}
	return
}

// func (s *Server) handleDisconnect(ctx context.Context, conn netpoll.Connection) {

// }
