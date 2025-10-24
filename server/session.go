package server

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/cloudwego/netpoll"
)

type Session struct {
	DeviceID         string
	Sender, Receiver *muxConn

	Ready       bool
	ReadyChan   chan struct{}
	DataChan    chan []byte
	Timeout     time.Duration
	LastActive  time.Time
	ConnectTime time.Time // 记录首次连接时间
	CloseNotify chan struct{}
	started     bool // 标记是否已启动
	// senderSeq   uint32
	receiverSeq uint32
	// mu          sync.RWMutex
	mu spinLock
}

func NewSession(deviceID string) *Session {
	ts := time.Now()
	return &Session{
		DeviceID: deviceID,
		// mu:       new(spinLock),
		ReadyChan:   make(chan struct{}),
		DataChan:    make(chan []byte, 100),
		Timeout:     30 * time.Second,
		LastActive:  ts,
		ConnectTime: ts,
		CloseNotify: make(chan struct{}),
	}
}

func (s *Session) SetSender(conn *muxConn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Sender != nil {
		slog.Error("Session sender already exists", "deviceID", s.DeviceID)
		return false
	}

	slog.Info("Session new sender connected ", "deviceID", s.DeviceID, "time", time.Now().Format(time.RFC3339))
	s.Sender = conn
	// 启动超时检测
	if !s.started {
		go s.StartDataForward()
		s.started = true
	}

	// // 发送连接成功状态
	// Encodex(conn.Writer(), &DatadPacket{
	// 	Header: &Header{
	// 		Cmd:   CMD_STATUS,
	// 		Order: 0,
	// 	},
	// 	Data: []byte{byte(STATUS_CONNECTED)},
	// })

	return s.checkReady()
}

func (s *Session) SetReceiver(conn *muxConn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Receiver != nil {
		slog.Debug("Session  receiver already exists", "deviceID", s.DeviceID)
		return false
	}

	slog.Info("Session new receiver connected  ", "deviceID", s.DeviceID, "time", time.Now().Format(time.RFC3339))
	s.Receiver = conn
	s.ConnectTime = time.Now() // 重置连接时间

	// 启动超时检测
	if !s.started {
		go s.StartDataForward()
		s.started = true
	}

	// 如果只有receiver连接，启动超时检测
	if s.Sender == nil {
		go func() {
			time.Sleep(s.Timeout)
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.Receiver != nil && s.Sender == nil {
				slog.Debug("Session  [receiver] timeout, no sender connected", "deviceId", s.DeviceID)
				s.handleSingleDisconnect(s.Receiver, STATUS_TIMEOUT)
				s.Receiver = nil
			}
		}()
	}

	// Encodex(conn.Writer(), &DatadPacket{
	// 	Header: &Header{
	// 		Cmd:   CMD_STATUS,
	// 		Order: 0,
	// 	},
	// 	Data: []byte{byte(STATUS_CONNECTED)},
	// })

	return s.checkReady()
}

func (s *Session) checkReady() bool {
	if s.Sender != nil && s.Receiver != nil {
		s.Ready = true
		close(s.ReadyChan)
		return true
	}
	return false
}

func (s *Session) StartDataForward() {
	// 启动超时检测
	go s.monitorTimeout()

	// 启动连接状态检测
	go s.monitorConnections()

	go func() {
		for {
			select {
			case data := <-s.DataChan:
				s.mu.Lock()
				// 处理指令码03
				if len(data) > 0 && data[0] == CMD_STATUS {
					status := data[1]
					if status == STATUS_CLOSED {
						slog.Debug("Session  [received] closed command", "deviceID", s.DeviceID)
						s.handleDisconnect(STATUS_CLOSED)
						s.mu.Unlock()
						return
					}
				}

				if s.Sender != nil {
					Writer := s.Sender.Writer()
					Writer.WriteBinary(data)
					if err := Writer.Flush(); err != nil {
						slog.Error("Failed to send data to sender", "error", err)
						s.handlePeerDisconnect(s.Sender, s.Receiver, STATUS_PEER_DISCONNECT)
					}
				}
				if s.Receiver != nil {
					Writer := s.Receiver.Writer()
					Writer.WriteBinary(data)
					if err := Writer.Flush(); err != nil {
						slog.Error("Failed to send data to receiver", "error", err)
						s.handlePeerDisconnect(s.Receiver, s.Sender, STATUS_PEER_DISCONNECT)
					}
				}
				s.mu.Unlock()
				s.LastActive = time.Now()

			case <-s.CloseNotify:
				return
			}
		}
	}()
}

// 启动超时检测
func (s *Session) monitorTimeout() {
	ticker := GetTicker(time.Second)
	defer ReleaseTicker(ticker)
	slog.Info("开始检查超时 ", "time", time.Now().Unix())
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			// slog.Info("定时检查超时 ", "time", time.Now().Unix())
			// 检查对端匹配超时
			if s.Sender != nil && s.Receiver == nil {
				elapsed := time.Since(s.ConnectTime)
				// slog.Info("Session  only [sender] connected", "deviceID", s.DeviceID, "elapsed", elapsed)
				if elapsed > s.Timeout {
					slog.Info("Session timeout: failed to match receiver", "deviceID", s.DeviceID)
					s.handleSingleDisconnect(s.Sender, STATUS_TIMEOUT)
					s.Sender = nil
				}
			} else if s.Receiver != nil && s.Sender == nil {
				elapsed := time.Since(s.ConnectTime)
				// slog.Info("Session  only [receiver] connected", "deviceID", s.DeviceID, "elapsed", elapsed)
				if elapsed > s.Timeout {
					slog.Info("Session timeout: failed to match sender", "deviceID", s.DeviceID)
					s.handleSingleDisconnect(s.Receiver, STATUS_TIMEOUT)
					s.Receiver = nil
				}
			}
			// // 检查数据传输超时
			// if s.Sender != nil && s.Receiver != nil {
			// 	if time.Since(s.LastActive) > s.Timeout {
			// 		slog.Info("Session timeout: no data transfer", "deviceID", s.DeviceID)
			// 		s.handleDisconnect(STATUS_TIMEOUT)
			// 	}
			// }
			s.mu.Unlock()

		case <-s.CloseNotify:
			slog.Info("Session closed", "deviceID", s.DeviceID)
			return
		}
	}
}

func (s *Session) handleSingleDisconnect(muxConn netpoll.Connection, status byte) {
	// writer := netpoll.NewLinkBuffer()
	Encodex(muxConn.Writer(), &DatadPacket{
		&Header{
			CMD_STATUS,
			0,
		},
		[]byte{byte(status)},
	})
	muxConn.Close()
	// muxConn.Put(func() (buf netpoll.Writer, isNil bool) {
	// 	return writer, false
	// })

	//muxConn.clear()
	// 清理session
	s.Ready = false
	close(s.CloseNotify)
}

func (s *Session) handleDisconnect(status byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// writer := s.
	// 	// 发送状态通知
	// 	Encodex(writer, &StatusPacket1{
	// 		Status:  status,
	// 		CmdCode: CMD_STATUS,
	// 		Order:   0,
	// 	})
	if s.Sender != nil {
		Encodex(s.Sender.Writer(),
			&DatadPacket{
				&Header{
					CMD_STATUS,
					0,
				},
				[]byte{byte(status)},
			})
		s.Sender.Close()
		// s.Sender.Put(func() (buf netpoll.Writer, isNil bool) {
		// 	return writer, false
		// })
		// s.Sender.clear()
		s.Sender = nil
	}

	if s.Receiver != nil {
		Encodex(s.Receiver.Writer(),
			&DatadPacket{
				&Header{
					CMD_STATUS,
					0,
				},
				[]byte{byte(status)},
			})
		s.Receiver.Close()
		// s.Receiver.Put(func() (buf netpoll.Writer, isNil bool) {
		// 	return writer, false
		// })
		// s.Sender.clear()
		s.Receiver = nil
	}

	// 清理session
	s.Ready = false
	close(s.CloseNotify)
}

func (s *Session) monitorConnections() {
	ticker := GetTicker(time.Second * 2)
	defer ReleaseTicker(ticker)

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			// 检测sender连接状态
			if s.Sender != nil && !s.Sender.IsActive() {
				slog.Error("Session  [sender] connection not active", "deviceID", s.DeviceID)
				s.handlePeerDisconnect(s.Sender, s.Receiver, STATUS_PEER_DISCONNECT)
			}
			// 检测receiver连接状态
			if s.Receiver != nil && !s.Receiver.IsActive() {
				slog.Error("Session  [receiver] connection not active", "deviceID", s.DeviceID)
				s.handlePeerDisconnect(s.Receiver, s.Sender, STATUS_PEER_DISCONNECT)
			}
			s.mu.Unlock()

		case <-s.CloseNotify:
			return
		}
	}
}

func (s *Session) handlePeerDisconnect(disconnectedConn, otherConn netpoll.Connection, status byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 关闭已断开的连接
	if disconnectedConn != nil {
		disconnectedConn.Close()
		if disconnectedConn == s.Sender {
			// s.Sender.Close()
			s.Sender = nil
		} else {
			// s.Receiver.Close()
			s.Receiver = nil
		}
	}

	// 通知另一端
	if otherConn != nil {
		writer := otherConn.Writer()
		// writer := netpoll.NewLinkBuffer()
		Encodex(writer, &DatadPacket{
			&Header{
				STATUS_PEER_DISCONNECT,
				0,
			},
			[]byte{byte(status)},
		})
		// otherConn.Put(func() (buf netpoll.Writer, isNil bool) {
		// 	return writer, false
		// })

		// otherConn.clear()
		otherConn.Close()
		if otherConn == s.Sender {
			s.Sender = nil
		} else {
			s.Receiver = nil
		}
	}

	// 清理session
	s.Ready = false
	close(s.CloseNotify)
}

func (s *Session) WaitReady() <-chan struct{} {
	return s.ReadyChan
}

// GetReceiverSeq returns the current receiver sequence number
func (s *Session) GetReceiverSeq() uint32 {
	return atomic.LoadUint32(&s.receiverSeq)
}

// IncrementReceiverSeq increments the receiver sequence number
func (s *Session) IncrementReceiverSeq() {
	atomic.AddUint32(&s.receiverSeq, 1)
}
