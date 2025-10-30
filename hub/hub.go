package hub

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"
	"unsafe"

	pl "tunnels/protocol"
	"tunnels/utils"

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
		netpoll.WithOnDisconnect(handleDisconnect),
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
//
//	func prepare(conn netpoll.Connection) (ctx context.Context) {
//		mc := newMuxConn(conn)
//		ctx = context.WithValue(context.Background(), ctxkey, mc)
//		return ctx
//	}
func (h *Hub) handleConnection2(ctx context.Context, conn netpoll.Connection) (err error) {
	defer conn.Close()
	reader := conn.Reader()
	writer := conn.Writer()
	for {
		r, err1 := reader.ReadBinary(reader.Len())
		if err1 != nil {
			return
		}

		_, err1 = writer.WriteBinary(r)
		if err1 != nil {
			return
		}
		writer.Flush()
		// cmd, err1 := reader.Peek(1)
		// if err1 != nil {
		// 	return
		// }
		// switch cmd[0] {
		// case pl.CMD_INIT:
		// }
		// pl.Decodex(reader)

	}
	// for {
	// 	pl.Encodex(reader, &pl.InitPacket{})
	// 	reader.Peek(1)
	// }

	// for {
	// 	a, _ := reader.ReadString(1)
	// 	fmt.Println(a)
	// 	reader.Release()
	// }
	//select {}

	// var initPacket pl.InitPacket
	// slog.Info("Processing initialization packet", "length", reader.Len())
	// if err = pl.ProcessRequest(reader, &initPacket); err != nil {
	// 	slog.Error("Initialization connection failed", "err", err)
	// 	return
	// }

	// session := h.sessionMar.GetOrCreate(initPacket.DeviceID)
	// // conn.Writer()
	// fmt.Printf("%+v\n", initPacket)
	// // 根据角色设置连接
	// var isReady bool
	// switch initPacket.Role {
	// case pl.ROLE_SENDER:
	// 	isReady = session.SetSender(conn)
	// 	slog.Info("[Sender] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	// case pl.ROLE_RECEIVER:
	// 	isReady = session.SetReceiver(conn)
	// 	slog.Info("[Receiver] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	// default:
	// 	return
	// }

	// seq := uint16(0)

	// //发送连接成功状态
	// response := &pl.DatadPacket{
	// 	Header: &pl.Header{
	// 		pl.CMD_STATUS,
	// 		seq,
	// 	},
	// 	Data: []byte{pl.STATUS_CONNECTED},
	// }
	// writer := conn.Writer()
	// if pl.Encodex(writer, response) != nil {
	// 	return
	// }
	// seq++

	// // 等待配对完成
	// if !isReady {
	// 	slog.Info("Waiting for peer connection .....", "deviceID", initPacket.DeviceID)
	// 	<-session.WaitReady()
	// 	slog.Info("Peer connection established .....", "deviceID", initPacket.DeviceID)
	// }

	// //发送传输状态
	// response.Order = seq
	// response.Data = []byte{pl.STATUS_READY}
	// if pl.Encodex(writer, response) != nil {
	// 	return
	// }
	// seq++

	// switch initPacket.Role {
	// case pl.ROLE_SENDER:
	// 	// 处理sender连接
	// 	for {
	// 		response, err1 := ForwardRequest(reader)
	// 		if err1 != nil {
	// 			// slog.Info("[sender] reader  Invalid packet ", "error", err1)
	// 			s.sessionMar.Remove(initPacket.DeviceID)
	// 			return
	// 		}

	// 		if response.Header.Cmd == CMD_DATA {
	// 			// slog.Info("[Received] data from sender ", "IP", conn.RemoteAddr(), "data", string(response.Data))

	// 			// 转发数据给receiver
	// 			if session.Receiver != nil {
	// 				// slog.Info("[Received]  Forwarding data to receiver from sender ", "deviceID", initPacket.DeviceID)
	// 				// Use receiver's sequence number for forwarded packets
	// 				slog.Info("[Sender]  Received  ", "Active", session.Receiver.IsActive())
	// 				receiverSeq := session.GetReceiverSeq()
	// 				response.Header.Order = uint16(receiverSeq)
	// 				Encodex(session.Receiver.Writer(), response)
	// 				session.IncrementReceiverSeq()
	// 			}
	// 			// slog.Info("[Sender]  Received  ", "Active", session.Receiver.IsActive())
	// 		}
	// 	}
	// case ROLE_RECEIVER:
	// 	for {
	// 		response, err1 := ForwardRequest(reader)
	// 		if err1 != nil {
	// 			s.sessionMar.Remove(initPacket.DeviceID)
	// 			return
	// 		}

	// 		if response.Header.Cmd == CMD_DATA {
	// 			// 转发数据给receiver
	// 			if session.Sender != nil {
	// 				// slog.Info("[Sender]  Forwarding data to send for device ", "deviceID", initPacket.DeviceID)
	// 				// Use receiver's sequence number for forwarded packets
	// 				slog.Info("[Sender]  Received  ", "Active", session.Sender.IsActive())
	// 				response.Header.Order = seq
	// 				Encodex(session.Sender.Writer(), response)
	// 			}
	// 			seq++
	// 		}
	// 	}
	// }

	select {}
	// for i := range 5 {
	// 	fmt.Println(i)
	// 	// handle connection
	// }
	return
}
func processRequest(reader netpoll.Reader) (data []byte, err error) {
	header, err := reader.ReadBinary(pl.HeaderSize)
	if err != nil {
		return
	}

	if header[0] != pl.CMD_INIT || header[0] != pl.CMD_STATUS || header[0] != pl.CMD_DATA {
		err = errors.New("invalid cmd code")
		return
	}

	dataL := binary.BigEndian.Uint16(header[3:5])
	if dataL <= 0 {
		return
	}

	payload, err := reader.ReadBinary(int(dataL) + 2)
	if err != nil {
		return
	}

	payloadL := len(payload)
	data = payload[:payloadL-2]

	// 计算并验证CRC
	if binary.BigEndian.Uint16(payload[payloadL-2:]) != utils.GetCRC16(slices.Concat(header, data)) {
		err = errors.New("CRC validation failed")
		return
	}

	err = reader.Release()
	return
}

func UnsafeSliceToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func (h *Hub) handleConnection(ctx context.Context, conn netpoll.Connection) (err error) {
	defer conn.Close()
	reader := conn.Reader()
	// writer := conn.Writer()
	for {
		req, err1 := pl.Decodex(reader)
		if err1 != nil {
			// fmt.Println(err1)
			return
		}

		switch req.Cmd {
		case pl.CMD_DATA:
			fmt.Println("data")
		case pl.CMD_INIT:
			fmt.Println("init")
			deviceCode := UnsafeSliceToString(bytes.TrimRight(req.Data[1:], "\x00"))
			//1.创建会话
			session := h.sessionMar.GetOrCreate(deviceCode)

			//1.1设置连接
			err = session.SetConner(req.Data[0], conn)

			// switch req.Data[0] {
			// case pl.ROLE_SENDER:
			// 	session.SetSender(conn)
			// 	// slog.InfoContext(ctx, "[Sender] connected for device", "deviceID", deviceCode)
			// case pl.ROLE_RECEIVER:
			// 	session.SetReceiver(conn)
			// 	slog.InfoContext(ctx, "[Receiver] connected for device", "deviceID", deviceCode)
			// default:
			// 	slog.Error("Invalid init packet: unexpected role", "deviceCode", deviceCode, "role", req.Data[0])
			// 	return
			// }
		default:
			fmt.Println("default")
			return
		}
	}

	// var initPacket pl.InitPacket
	// if err = pl.ProcessRequest(reader, &initPacket); err != nil {
	// 	slog.Error("Initialization connection failed", "err", err)
	// 	return
	// }

	// session := h.sessionMar.GetOrCreate(initPacket.DeviceID)
	// var isReady bool
	// switch initPacket.Role {
	// case pl.ROLE_SENDER:
	// 	session.SetSender(conn)
	// 	slog.Info("[Sender] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	// case pl.ROLE_RECEIVER:
	// 	session.SetReceiver(conn)
	// 	slog.Info("[Receiver] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	// default:
	// 	return
	// }

	// seq := uint16(0)

	// //发送连接成功状态
	// response := &pl.DatadPacket{
	// 	Header: &pl.Header{
	// 		pl.CMD_STATUS,
	// 		seq,
	// 	},
	// 	Data: []byte{pl.STATUS_CONNECTED},
	// }
	// writer := conn.Writer()
	// if pl.Encodex(writer, response) != nil {
	// 	return
	// }
	// seq++

	// // 等待配对完成
	// if !isReady {
	// 	slog.Info("Waiting for peer connection .....", "deviceID", initPacket.DeviceID)
	// 	<-session.WaitReady()
	// 	slog.Info("Peer connection established .....", "deviceID", initPacket.DeviceID)
	// }

	// writer := conn.Writer()
	// for {

	// 	// r, err1 := reader.ReadBinary(reader.Len())
	// 	// if err1 != nil {
	// 	// 	return
	// 	// }

	// 	// cmd, err1 := reader.Peek(1)
	// 	// if err1 != nil {
	// 	// 	return
	// 	// }
	// 	// switch cmd[0] {
	// 	// case pl.CMD_INIT:
	// 	// }
	// 	// pl.Decodex(reader)

	// }
	// for {
	// 	pl.Encodex(reader, &pl.InitPacket{})
	// 	reader.Peek(1)
	// }

	// for {
	// 	a, _ := reader.ReadString(1)
	// 	fmt.Println(a)
	// 	reader.Release()
	// }
	//select {}

	// var initPacket pl.InitPacket
	// slog.Info("Processing initialization packet", "length", reader.Len())
	// if err = pl.ProcessRequest(reader, &initPacket); err != nil {
	// 	slog.Error("Initialization connection failed", "err", err)
	// 	return
	// }

	// session := h.sessionMar.GetOrCreate(initPacket.DeviceID)
	// // conn.Writer()
	// fmt.Printf("%+v\n", initPacket)
	// // 根据角色设置连接
	// var isReady bool
	// switch initPacket.Role {
	// case pl.ROLE_SENDER:
	// 	isReady = session.SetSender(conn)
	// 	slog.Info("[Sender] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	// case pl.ROLE_RECEIVER:
	// 	isReady = session.SetReceiver(conn)
	// 	slog.Info("[Receiver] connected for device", "deviceID", initPacket.DeviceID, "IP", conn.RemoteAddr().String())
	// default:
	// 	return
	// }

	// seq := uint16(0)

	// //发送连接成功状态
	// response := &pl.DatadPacket{
	// 	Header: &pl.Header{
	// 		pl.CMD_STATUS,
	// 		seq,
	// 	},
	// 	Data: []byte{pl.STATUS_CONNECTED},
	// }
	// writer := conn.Writer()
	// if pl.Encodex(writer, response) != nil {
	// 	return
	// }
	// seq++

	// // 等待配对完成
	// if !isReady {
	// 	slog.Info("Waiting for peer connection .....", "deviceID", initPacket.DeviceID)
	// 	<-session.WaitReady()
	// 	slog.Info("Peer connection established .....", "deviceID", initPacket.DeviceID)
	// }

	// //发送传输状态
	// response.Order = seq
	// response.Data = []byte{pl.STATUS_READY}
	// if pl.Encodex(writer, response) != nil {
	// 	return
	// }
	// seq++

	// switch initPacket.Role {
	// case pl.ROLE_SENDER:
	// 	// 处理sender连接
	// 	for {
	// 		response, err1 := ForwardRequest(reader)
	// 		if err1 != nil {
	// 			// slog.Info("[sender] reader  Invalid packet ", "error", err1)
	// 			s.sessionMar.Remove(initPacket.DeviceID)
	// 			return
	// 		}

	// 		if response.Header.Cmd == CMD_DATA {
	// 			// slog.Info("[Received] data from sender ", "IP", conn.RemoteAddr(), "data", string(response.Data))

	// 			// 转发数据给receiver
	// 			if session.Receiver != nil {
	// 				// slog.Info("[Received]  Forwarding data to receiver from sender ", "deviceID", initPacket.DeviceID)
	// 				// Use receiver's sequence number for forwarded packets
	// 				slog.Info("[Sender]  Received  ", "Active", session.Receiver.IsActive())
	// 				receiverSeq := session.GetReceiverSeq()
	// 				response.Header.Order = uint16(receiverSeq)
	// 				Encodex(session.Receiver.Writer(), response)
	// 				session.IncrementReceiverSeq()
	// 			}
	// 			// slog.Info("[Sender]  Received  ", "Active", session.Receiver.IsActive())
	// 		}
	// 	}
	// case ROLE_RECEIVER:
	// 	for {
	// 		response, err1 := ForwardRequest(reader)
	// 		if err1 != nil {
	// 			s.sessionMar.Remove(initPacket.DeviceID)
	// 			return
	// 		}

	// 		if response.Header.Cmd == CMD_DATA {
	// 			// 转发数据给receiver
	// 			if session.Sender != nil {
	// 				// slog.Info("[Sender]  Forwarding data to send for device ", "deviceID", initPacket.DeviceID)
	// 				// Use receiver's sequence number for forwarded packets
	// 				slog.Info("[Sender]  Received  ", "Active", session.Sender.IsActive())
	// 				response.Header.Order = seq
	// 				Encodex(session.Sender.Writer(), response)
	// 			}
	// 			seq++
	// 		}
	// 	}
	// }

	// for i := range 5 {
	// 	fmt.Println(i)
	// 	// handle connection
	// }
	return
}

func handleDisconnect(ctx context.Context, conn netpoll.Connection) {
	// mc := ctx.Value(ctxkey).(*muxConn)
	slog.Info("关闭连接")
}
