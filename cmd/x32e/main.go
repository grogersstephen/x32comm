package main

// x32 Emulator

import (
	"encoding/hex"
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

	// b, err := hex.DecodeString("2f63682f32312f6d69782f6661646572000000002c66000000000000")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	b, err := hex.DecodeString("2f6d65746572732f360000002c6200000000001404000000fd1d2137fdff7f3f0000803f6ebbd534")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(b), string(b))

	_, err = udpServer.WriteTo(b[:], addr)
	if err != nil {
		fmt.Println("write to", err)
	}
}
