package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"slices"
	"sync/atomic"
	"time"
	pl "tunnels/protocol"
	"tunnels/utils"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9085")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to server at localhost:9085")
	conn.Write(pl.Encode(pl.DatadPacket{
		Header: pl.Header{
			Cmd:   pl.CMD_INIT,
			Order: 0,
		},
		Data: slices.Concat([]byte{pl.ROLE_SENDER}, []byte("7062450000771987999")),
		// DeviceID: "7062450000771987999",
		// Role:     pl.ROLE_SENDER,
		// Header: &pl.Header{
		// 	Cmd:   pl.CMD_INIT,
		// 	Order: 0,
		// },
	}))
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
	var ready uint32
	ctx, cancel := context.WithCancel(context.Background())
	for {
		select {
		case mgs := <-readcl:
			switch mgs.Cmd {
			// case pl.CMD_DATA:
			// 	if atomic.LoadUint32(&ready) != 1 {
			// 		return
			// 	}

			// 	conn.Write(pl.Encode(pl.DatadPacket{
			// 		Header: pl.Header{
			// 			Cmd:   pl.CMD_DATA,
			// 			Order: 0,
			// 		},
			// 		Data: []byte(time.Now().Format("2006-01-02 15:04:05")),
			// 	}))
			case pl.CMD_STATUS:
				if mgs.Data[0] == pl.STATUS_CONNECTED {
					atomic.StoreUint32(&ready, 1)
				} else if mgs.Data[0] == pl.STATUS_READY && atomic.LoadUint32(&ready) == 1 {
					for {
						select {
						case <-ctx.Done():
							return
						default:
							conn.Write(pl.Encode(pl.DatadPacket{
								Header: pl.Header{
									Cmd:   pl.CMD_DATA,
									Order: 0,
								},
								Data: []byte(time.Now().Format("2006-01-02 15:04:05")),
							}))
							time.Sleep(time.Second * 1)
						}
					}
				} else {
					cancel()
					fmt.Printf("关闭连接, 状态%d\n", mgs.Data[0])
					return
				}

			}
		}
	}
}

func ReceiveData(cl chan *pl.DatadPacket, plaod []byte) (err error) {
	// fmt.Printf("%x\n", plaod)
	index := 5
LOOP:
	if len(plaod) < index {
		return
	}
	// fmt.Println(333333)
	// cmd := plaod[0]
	l := int(binary.BigEndian.Uint16(plaod[3:index]))
	// fmt.Printf("数据长度:%d\n", l)
	data := plaod[index : index+l]
	// fmt.Printf("data：%x\n", data)
	crc16 := binary.BigEndian.Uint16(plaod[index+l : index+l+2])

	if crc16 != utils.GetCRC16(plaod[:index+l]) {
		fmt.Println("crc 不相等")
		err = errors.New("crc16 inval")
		return
	}

	cl <- &pl.DatadPacket{
		Header: pl.Header{
			Cmd:   plaod[0],
			Order: binary.BigEndian.Uint16(plaod[1:3]),
		},
		Data: data,
	}

	plaod = plaod[index+l+2:]

	goto LOOP
}
