package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	pl "tunnels/protocol"
)

func main() {
	conn, err := net.Dial("tcp", "192.168.22.25:9703")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to server at localhost:9703")

	buf := bytes.NewBuffer([]byte{})
	//字段数量
	buf.WriteByte(2)

	// 辅助函数：添加带标记和长度的数据到缓冲区
	appendData := func(tag byte, data []byte) {
		buf.WriteByte(tag)
		binary.Write(buf, binary.BigEndian, uint16(len(data))) //小端两字节
		buf.Write(data)
	}
	//角色字段
	appendData(0x00, []byte{pl.ROLE_RECEIVER})
	appendData(0x01, []byte("7062450000771987999"))
	conn.Write(pl.Encode(pl.DatadPacket{
		Cmd:  pl.CMD_INIT,
		Data: buf.Bytes(),
		// DeviceID: "7062450000771987999",
		// Role:     pl.ROLE_SENDER,
		// Header: &pl.Header{
		// 	Cmd:   pl.CMD_INIT,
		// 	Order: 0,
		// },
	}))
	buf.Reset()
	readcl := make(chan *pl.DatadPacket, 10)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Fatalf("Failed to read response: %v", err)
			}
			ReceiveData(readcl, buf[:n])
			// pl.DecodeData(buf[:n])
			// fmt.Println(string(buf[:n]) + "/8")
			// fmt.Printf("%v\n", buf[:n])
		}
	}()
	// var ready uint32
	// ctx, cancel := context.WithCancel(context.Background())
	for {
		select {
		case mgs := <-readcl:
			switch mgs.Cmd {
			case pl.CMD_DATA:
				fmt.Println("接收数据:" + string(mgs.Data))
			case pl.CMD_STATUS:
				if mgs.Data[0] == pl.STATUS_CONNECTED {
					fmt.Println("已成功连接服务器，等待接发送端。。。。。。。。。。。。。。。。")
				} else if mgs.Data[0] == pl.STATUS_READY {
					fmt.Println("已就绪，开始接收数据。。。。。。。。。。。。。。。")
				} else if mgs.Data[0] == pl.STATUS_PEER_DISCONNECT {
					fmt.Println("发送端关闭了")
				}
				// } else {
				// 	cancel()
				// 	fmt.Printf("关闭连接, 状态%d\n", mgs.Data[0])
				// 	return
				// }

			}
		}
	}
}

func ReceiveData(cl chan *pl.DatadPacket, plaod []byte) (err error) {
	// fmt.Printf("%x\n", plaod)
	index := 3
LOOP:
	if len(plaod) < index {
		return
	}
	// fmt.Println(333333)
	// cmd := plaod[0]
	l := int(binary.BigEndian.Uint16(plaod[1:index]))
	// fmt.Printf("数据长度:%d\n", l)
	data := plaod[index : index+l]
	fmt.Printf("%x\n", data)
	// fmt.Printf("data：%x\n", data)
	// crc16 := binary.BigEndian.Uint16(plaod[index+l : index+l+2])

	// if crc16 != utils.GetCRC16(plaod[:index+l]) {
	// 	fmt.Println("crc 不相等")
	// 	err = errors.New("crc16 inval")
	// 	return
	// }

	cl <- &pl.DatadPacket{
		Cmd:  plaod[0],
		Data: data,
	}

	plaod = plaod[index+l:]

	goto LOOP
}
