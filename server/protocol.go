package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"tunnels/utils"

	"github.com/cloudwego/netpoll"
)

const (
	HeaderSize = 6 // 指令码 + 通讯ID + 数据长度 + CRC16
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

// 协议头
type Header struct {
	Cmd byte   // 指令码
	Seq uint16 // 通讯ID
	// Length uint16 // 数据长度
	CRC uint16 // CRC16校验
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

type StatusPacket1 struct {
	Status  uint8  // 连接状态
	CmdCode uint8  //指令码
	Order   uint16 // 业务流水号
}

// 编码数据包
func EncodeData(data []byte) []byte {
	buf := new(bytes.Buffer)
	// 写入数据长度
	binary.Write(buf, binary.LittleEndian, uint16(len(data)))
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
func Encode(cmd byte, seq uint16, data []byte) []byte {
	// b := netpoll.NewLinkBuffer(len(data) + HeaderSize)

	buf := new(bytes.Buffer)
	// 写入指令码
	buf.WriteByte(cmd)
	// 写入通讯ID
	binary.Write(buf, binary.LittleEndian, seq)
	// 写入数据包
	encodedData := EncodeData(data)
	buf.Write(encodedData)

	// Get packet data before CRC
	packetData := buf.Bytes()
	// Calculate CRC on packet data
	crc := utils.GetCRC16(packetData)
	// Write CRC
	binary.Write(buf, binary.LittleEndian, crc)
	// log.Printf("encode crc16: %v (data length: %d)", crc, len(packetData))

	return buf.Bytes()
}
func ProcessRequest(reader netpoll.Reader, h *InitPacket) (err error) {
	if reader.Len() < 6 {
		return errors.New("invalid packet length")
	}

	r2, err := reader.Peek(reader.Len() - 2)
	if err != nil {
		err = errors.New("reader cmd invalid 1")
		return
	}

	//计算crc16
	calculatedCRC := utils.GetCRC16(r2)

	// 读取指令码
	cmd, err := reader.ReadByte()
	if err != nil {
		err = errors.New("reader cmd invalid 2")
		return
	}
	// 读取通讯ID (2 bytes)
	cid, err := reader.Next(2)
	if err != nil {
		err = errors.New("reader cmd invalid")
		return
	}
	seq := binary.LittleEndian.Uint16(cid)

	// 指令数据内容长度(2 bytes)
	dlen, err := reader.Next(2)
	if err != nil {
		err = errors.New("dlen invalid")
		return
	}
	dataLen := binary.LittleEndian.Uint16(dlen)
	// fmt.Printf("数据长度：%d\n", dataLen)
	//读取数据
	data, err := reader.Next(int(dataLen))
	if err != nil {
		err = errors.New("data  invalid")
		return
	}

	//读取crc16
	crcBytes, err := reader.Next(2)
	if err != nil {
		err = errors.New("CRC16 invalid")
		return
	}
	crc := binary.LittleEndian.Uint16(crcBytes)

	// 计算并验证CRC
	if crc != calculatedCRC {
		err = errors.New("CRC validation failed")
		log.Printf("received crc mismatch:%X,calculated crc mismatch:%X", crc, calculatedCRC)
		return
	}

	h = &InitPacket{
		&Header{
			Cmd: cmd,
			Seq: seq,
			CRC: crc,
		},
		uint16(data[0]),
		string(bytes.TrimRight(data[1:], "\x00")),
	}
	err = reader.Release()
	return
}

// decode
// func Decodex(reader netpoll.Reader, h *Header) (err error) {
// 	if reader.Len() < 6 {
// 		return errors.New("invalid packet length")
// 	}

// 	r2, err := reader.Peek(reader.Len() - 2)
// 	if err != nil {
// 		err = errors.New("reader cmd invalid 1")
// 		return
// 	}

// 	//计算crc16
// 	calculatedCRC := utils.GetCRC16(r2)

// 	// 读取指令码
// 	h.Cmd, err = reader.ReadByte()
// 	if err != nil {
// 		err = errors.New("reader cmd invalid 2")
// 		return
// 	}
// 	// 读取通讯ID (2 bytes)
// 	cid, err := reader.Next(2)
// 	if err != nil {
// 		err = errors.New("reader cmd invalid")
// 		return
// 	}
// 	h.Seq = binary.LittleEndian.Uint16(cid)

// 	// 指令数据内容长度(2 bytes)
// 	dlen, err := reader.Next(2)
// 	if err != nil {
// 		err = errors.New("dlen invalid")
// 		return
// 	}
// 	dataLen := binary.LittleEndian.Uint16(dlen)
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
// 	h.CRC = binary.LittleEndian.Uint16(crcBytes)

// 	// 计算并验证CRC
// 	if h.CRC != calculatedCRC {
// 		err = errors.New("CRC validation failed")
// 		log.Printf("received crc mismatch:%X,calculated crc mismatch:%X", h.CRC, calculatedCRC)
// 		return
// 	}

// 	// 解码数据包
// 	// b, err := DecodeData(data)
// 	// if err != nil {
// 	// 	return
// 	// }

// 	h.Data = &InitPacket{
// 		Role:     uint16(data[0]),
// 		DeviceID: string(bytes.TrimRight(data[1:], "\x00")),
// 	}
// 	err = reader.Release()
// 	return
// }

func Encodex(writer netpoll.Writer, d *StatusPacket1) {
	//指令编码1字节 + 业务流水号2字节 + 指令数据内容长度2字节 + 数据 +CRC2字节
	data := []byte{d.Status}
	length := len(data)
	header, _ := writer.Malloc(1 + 2 + 2 + length + 2)
	//指令编码
	header = append(header, d.CmdCode)
	//业务流水号
	header = binary.LittleEndian.AppendUint16(header, uint16(d.Order))
	// 指令数据内容长度
	header = binary.LittleEndian.AppendUint16(header, uint16(length))
	//数据
	header = append(header, data...)

	//CRC16
	binary.LittleEndian.AppendUint16(header, utils.GetCRC16(header))
	writer.Flush()
	return
}

// 解码
func Decode(data []byte) (Header, []byte, error) {
	if len(data) < 5 {
		return Header{}, nil, errors.New("invalid packet length")
	}

	// 读取指令码
	header := Header{
		Cmd: data[0],
	}

	// 读取通讯ID (2 bytes)
	header.Seq = binary.LittleEndian.Uint16(data[1:3])

	// 读取CRC16 (last 2 bytes)
	receivedCRC := binary.LittleEndian.Uint16(data[len(data)-2:])

	// 计算并验证CRC
	calculatedCRC := utils.GetCRC16(data[:len(data)-2])
	if receivedCRC != calculatedCRC {
		log.Printf("received crc mismatch:%X,calculated crc mismatch:%X", receivedCRC, calculatedCRC)
		return Header{}, nil, errors.New("CRC validation failed")
	}

	// 解码数据包
	payload, err := DecodeData(data[3 : len(data)-2])
	if err != nil {
		return Header{}, nil, err
	}

	return header, payload, nil
}
