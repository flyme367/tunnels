package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
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
	conn.Write(pl.Encode(pl.InitPacket{
		DeviceID: "7062450000771987999",
		Role:     pl.ROLE_SENDER,
		Header: &pl.Header{
			Cmd:   pl.CMD_INIT,
			Order: 0,
		},
	}))
	readcl := make(chan *pl.DatadPacket, 10)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Fatalf("Failed to read response: %v", err)
			}
			pl.DecodeData(buf[:n])
			// fmt.Println(string(buf[:n]) + "/8")
			// fmt.Printf("%v\n", buf[:n])
		}
	}()

	for {
		select {
		case mgs := <-readcl:
			fmt.Printf("%+v\n", mgs)
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
