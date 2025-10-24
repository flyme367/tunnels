package main

import (
	"fmt"
	"log"
	"net"
	pl "tunnels/protocol"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9085")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to server at localhost:9085")
	initPacket := pl.InitPacket{
		Header: &pl.Header{
			Cmd:   pl.CMD_INIT,
			Order: 0,
		},
		DeviceID: "7062450000771987999",
		Role:     pl.ROLE_SENDER,
	}
	initData := pl.Encode(initPacket)
	conn.Write(initData)
	buf := make([]byte, 1024)
	for {
		// _, err := conn.Write([]byte("Hello, server!"))
		// if err != nil {
		// 	log.Fatalf("Failed to send message: %v", err)
		// }
		// log.Println("Sent message to server")

		// buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read response: %v", err)
		}
		fmt.Println(string(buf[:n]))
		// log.Printf("Received response from server: %s", string(buf[:n]))
	}
}
