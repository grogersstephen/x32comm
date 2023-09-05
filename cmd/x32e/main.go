package main

// x32 Emulator

import (
	"fmt"
	"log"
	"net"
)

func main() {
	// listen to incoming udp packets
	udpServer, err := net.ListenPacket("udp", ":1053")
	if err != nil {
		log.Fatal(err)
	}
	defer udpServer.Close()

	for {
		buf := make([]byte, 1024)
		_, addr, err := udpServer.ReadFrom(buf)
		if err != nil {
			fmt.Println("err", err)
			continue
		}
		go response(udpServer, addr, buf)
	}

}

func response(udpServer net.PacketConn, addr net.Addr, buf []byte) {

	_, err := udpServer.WriteTo(buf[:], addr)
	if err != nil {
		fmt.Println("write to", err)
	}
}
