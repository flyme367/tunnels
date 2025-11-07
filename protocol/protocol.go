package hub

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"tunnels/utils"

	"github.com/cloudwego/netpoll"
)

const (
	HeaderSize = 5 // 指令码 + 通讯ID + 数据长度 + CRC16
)

// 指令码
const (
	CMD_INIT   = iota + 0x01 // 初始化连接
	CMD_STATUS               // 状态信息
	CMD_DATA                 // 数据包
)

// 连接角色
const (
	ROLE_SENDER   = iota + 0x01 // 发送端
	ROLE_RECEIVER               // 接收端
)

// 连接状态
const (
	STATUS_CONNECTED       = iota + 0x01 // 连接成功
	STATUS_READY                         // 传输就绪
	STATUS_CLOSED                        // 连接终止
	STATUS_TIMEOUT                       // 连接超时
	STATUS_PEER_DISCONNECT               // 对端异常终止
	//传输等待
)

type ForwardCtx struct{}
type SessionCtx struct{}

// 协议头
type Header struct {
	Cmd byte // 指令码
	// Order uint16 // 通讯ID
	// Length uint16 // 数据长度
	// CRC uint16 // CRC16校验
	// Data *InitPacket
}

// 初始化连接包
type InitPacket struct {
	Header   *Header
	Role     uint16 // 连接角色
	DeviceID string // 设备标识
}

// 状态包
type StatusPacket struct {
	Status byte // 连接状态
}

// type StatusPacket1 struct {
// 	Status  uint8  // 连接状态
// 	CmdCode uint8  //指令码
// 	Order   uint16 // 业务流水号
// }

type DatadPacket struct {
	Cmd  byte // 指令码
	Data []byte
}

// 编码数据包
func EncodeData(data []byte) []byte {
	buf := new(bytes.Buffer)
	// 写入数据长度
	binary.Write(buf, binary.BigEndian, uint16(len(data)))
	// 写入数据内容
	buf.Write(data)
	return buf.Bytes()
}

// 解码数据包
func DecodeData(data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, errors.New("invalid data length")
	}
	fmt.Printf("data: %x\n", data)
	// 读取数据长度
	length := int(data[0])
	if len(data) < 1+length {
		return nil, errors.New("incomplete data packet")
	}

	// 返回数据内容
	//data[2 : 1+length]
	return data[2:], nil
}

// 编码
func Encode(req DatadPacket) []byte {
	// b := netpoll.NewLinkBuffer(len(data) + HeaderSize)

	buf := new(bytes.Buffer)
	// 写入指令码
	buf.WriteByte(req.Cmd)
	// // 写入通讯ID
	// binary.Write(buf, binary.BigEndian, req.Header.Order)
	// 写入数据包
	// encodedData := make([]byte, 1+len(req.DeviceID))
	// encodedData[0] = byte(req.Role)
	// copy(encodedData[1:], req.DeviceID)
	//写入数据长度
	binary.Write(buf, binary.BigEndian, uint16(len(req.Data)))
	buf.Write(req.Data)

	// Get packet data before CRC
	// packetData := buf.Bytes()
	// // Calculate CRC on packet data
	// crc := utils.GetCRC16(packetData)
	// // Write CRC
	// binary.Write(buf, binary.BigEndian, crc)
	// log.Printf("encode crc16: %v (data length: %d)", crc, len(packetData))
	bufs := buf.Bytes()
	fmt.Printf("encode: %x\n", bufs)
	return bufs
}

// func Decodexx(reader netpoll.Reader) (h *DatadPacket, err error) {
// 	// if reader.Len() < 7 {
// 	// 	err = errors.New("invalid packet length")
// 	// 	return
// 	// }

// 	r2, err := reader.Peek(reader.Len() - 2)
// 	if err != nil {
// 		err = errors.New("reader cmd invalid 1")
// 		return
// 	}

// 	//计算crc16
// 	calculatedCRC := utils.GetCRC16(r2)

// 	// fmt.Println("calculatedCRC:", calculatedCRC)
// 	// 读取指令码
// 	cmd, err := reader.ReadByte()
// 	if err != nil {
// 		err = errors.New("reader cmd invalid 2")
// 		return
// 	}
// 	// 读取通讯ID (2 bytes)
// 	cid, err := reader.Next(2)
// 	if err != nil {
// 		err = errors.New("reader cid invalid")
// 		return
// 	}
// 	seq := binary.BigEndian.Uint16(cid)

// 	// 指令数据内容长度(2 bytes)
// 	dlen, err := reader.Next(2)
// 	if err != nil {
// 		err = errors.New("dlen invalid")
// 		return
// 	}
// 	dataLen := binary.BigEndian.Uint16(dlen)
// 	// fmt.Printf("数据长度：%d\n", dataLen)
// 	//读取数据
// 	data, err := reader.Next(int(dataLen))
// 	if err != nil {
// 		err = errors.New("data  invalid")
// 		return
// 	}

// 	//读取crc16
// 	crcBytes, err := reader.Next(2)
// 	if err != nil {
// 		err = errors.New("CRC16 invalid")
// 		return
// 	}
// 	crc := binary.BigEndian.Uint16(crcBytes)

// 	// 计算并验证CRC
// 	if crc != calculatedCRC {
// 		err = errors.New("CRC validation failed")
// 		return
// 	}

// 	h = &DatadPacket{
// 		&Header{
// 			cmd,
// 			seq,
// 		},
// 		data,
// 	}
// 	return
// }

func ReceiveData(reader netpoll.Reader, malloc []byte) (it iter.Seq[*DatadPacket], err error) {
	it = func(yield func(*DatadPacket) bool) {
		for {
			// // 检查是否有足够的数据进行读取
			// if reader.Len() < HeaderSize {
			// 	return
			// }

			// 读取指令码
			cmd, err := reader.ReadByte()
			if err != nil {
				return
			}

			// 检查指令码是否有效
			if cmd != CMD_DATA && cmd != CMD_STATUS && cmd != CMD_INIT {
				// 尝试恢复读取位置
				return
			}

			// 确保有足够的数据读取头部剩余部分
			if reader.Len() < HeaderSize-1 {
				// 将指令码放回并返回
				// 这里简化处理，实际可能需要更复杂的恢复机制
				return
			}

			// 读取头部剩余部分
			header, err := reader.ReadBinary(HeaderSize - 1)
			if err != nil {
				return
			}

			// 获取数据长度
			dataL := binary.BigEndian.Uint16(header[2:4])

			// 确保有足够的数据读取数据和CRC
			if reader.Len() < int(dataL)+2 {
				// 数据不完整，放回已读取的数据
				return
			}

			// 读取数据和CRC
			payload, err := reader.ReadBinary(int(dataL) + 2)
			if err != nil {
				return
			}

			// 构造用于CRC计算的完整数据
			malloc[0] = cmd
			copy(malloc[1:HeaderSize], header)
			payloadL := len(payload)
			copy(malloc[HeaderSize:HeaderSize+int(dataL)], payload[:payloadL-2])

			// 计算并验证CRC
			if binary.BigEndian.Uint16(payload[payloadL-2:]) != utils.GetCRC16(malloc[:HeaderSize+int(dataL)]) {
				// CRC校验失败，跳过这个包继续处理下一个
				continue
			}

			// 构造数据包
			h := &DatadPacket{

				Cmd: cmd,

				Data: payload[:payloadL-2],
			}

			// 通过 yield 返回数据包
			if !yield(h) {
				return
			}
		}
	}
	return
}

func Decodex(reader netpoll.Reader, req *DatadPacket, malloc []byte) (err error) {
	defer reader.Release()
	malloc[0], err = reader.ReadByte()
	if err != nil {
		// fmt.Println("报错了？")
		// err = reader.Release()
		return
	}

	if malloc[0] != CMD_DATA &&
		malloc[0] != CMD_STATUS &&
		malloc[0] != CMD_INIT {
		err = errors.New("invalid cmd code")
		return
	}

	header, err := reader.ReadBinary(HeaderSize - 1)
	if err != nil {
		return
	}

	dataL := binary.BigEndian.Uint16(header[2:4])
	if dataL <= 0 {
		return
	}

	payload, err := reader.ReadBinary(int(dataL) + 2)
	if err != nil {
		return
	}

	payloadL := len(payload)
	copy(malloc[1:HeaderSize], header)
	copy(malloc[HeaderSize:HeaderSize+dataL], payload[:payloadL-2])

	// 计算并验证CRC
	if binary.BigEndian.Uint16(payload[payloadL-2:payloadL]) != utils.GetCRC16(malloc[:HeaderSize+dataL]) {
		err = errors.New("CRC validation failed")
		return
	}
	req.Cmd = malloc[0]
	req.Data = malloc[:HeaderSize+dataL]
	return
}

func Encodex(writer netpoll.Writer, d *DatadPacket) (err error) {
	//指令编码1字节 + 业务流水号2字节 + 指令数据内容长度2字节 + 数据 +CRC2字节
	// data := []byte{d.Status}
	length := len(d.Data)
	header, _ := writer.Malloc(1 + 2 + length)
	//指令编码
	header[0] = d.Cmd
	// //业务流水号
	// binary.BigEndian.PutUint16(header[1:3], d.Order)
	// 指令数据内容长度
	binary.BigEndian.PutUint16(header[1:3], uint16(length))
	// fmt.Printf("data length: %d\n", length)
	copy(header[3:3+length], d.Data)
	// fmt.Printf("header: %X\n", header[5:5+length])

	// //CRC16
	// crc16 := make([]byte, 2)
	// // fmt.Printf("writer: %x\n", header[:writer.MallocLen()])
	// // fmt.Printf("writer Len: %x\n", writer.MallocLen())
	// binary.BigEndian.PutUint16(crc16, utils.GetCRC16(header))
	// // fmt.Println(writer.MallocLen())
	// //数据
	// // writer.WriteBinary(data)
	// // fmt.Printf("crc16: %x\n", crc16)
	// // fmt.Printf("crc16: %d\n", binary.BigEndian.Uint16(crc16))
	// writer.WriteBinary(crc16)
	return writer.Flush()
}

// 解码
// func Decode(data []byte) (Header, []byte, error) {
// 	if len(data) < 5 {
// 		return Header{}, nil, errors.New("invalid packet length")
// 	}

// 	// 读取指令码
// 	header := Header{
// 		Cmd: data[0],
// 	}

// 	// 读取通讯ID (2 bytes)
// 	header.Seq = binary.BigEndian.Uint16(data[1:3])

// 	// 读取CRC16 (last 2 bytes)
// 	receivedCRC := binary.BigEndian.Uint16(data[len(data)-2:])

// 	// 计算并验证CRC
// 	calculatedCRC := utils.GetCRC16(data[:len(data)-2])
// 	if receivedCRC != calculatedCRC {
// 		log.Printf("received crc mismatch:%X,calculated crc mismatch:%X", receivedCRC, calculatedCRC)
// 		return Header{}, nil, errors.New("CRC validation failed")
// 	}

// 	// 解码数据包
// 	payload, err := DecodeData(data[3 : len(data)-2])
// 	if err != nil {
// 		return Header{}, nil, err
// 	}

// 	return header, payload, nil
// }

// func DecodeData(data []byte) ([]byte, error) {
// 	if len(data) < 1 {
// 		return nil, errors.New("invalid packet length")
// 	}

// 	// 读取数据内容长度
// 	dataLen := int(data[0])

// 	// 读取数据
// 	if len(data) < dataLen+1 {
// 		return nil, errors.New("invalid packet length")
// 	}

// 	return data[1 : dataLen+1], nil

// }
// func ForwardRequest(reader netpoll.Reader) (req *DatadPacket, err error) {
// 	if req, err = Decodex(reader); err != nil {
// 		return
// 	}
// 	err = reader.Release()
// 	return
// }
