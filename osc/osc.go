package osc

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"time"
)

type OSC struct {
	Destination *net.UDPAddr
	Client      *net.UDPAddr
	Conn        *net.UDPConn
}

func (osc *OSC) Dial() error {
	var err error
	fmt.Fprintf(os.Stderr, "osc.Destination: %v:%v\n", osc.Destination.IP, osc.Destination.Port)
	osc.Conn, err = net.DialUDP("udp", osc.Client, osc.Destination)
	fmt.Fprintf(os.Stderr, "osc.Conn local addr: %s\n", osc.Conn.LocalAddr().String())
	return err
}

func (osc *OSC) SendString(s string) error {
	// Sends a message using the Conn from net.DialUDP
	byt := []byte(s)
	// If we have written tilde, conver to zero bytes
	for i := range byt {
		if string(byt[i]) == "~" {
			byt[i] = 0
		}
	}
	// Write the bytes to the connection
	_, err := osc.Conn.Write(byt)
	return err
}

func (osc *OSC) Send(msg Message) error {
	if msg.Packet.Len() == 0 {
		err := msg.MakePacket()
		if err != nil {
			return err
		}
	}
	// Sends a message using the Conn from net.DialUDP
	_, err := osc.Conn.Write(msg.Packet.Bytes())
	return err
}

func (osc *OSC) Receive(wait time.Duration) (Message, error) {
	// Waits for and reads a message from Conn from net.DialUDP
	//    wait is wait time in milliseconds
	var (
		msg Message
		err error
		wg  sync.WaitGroup
	)

	byt := make([]byte, 8192)

	fmt.Fprintf(os.Stderr, "Waiting for Reply...\n")
	wg.Add(1) // Add one count to waitgroup
	go func(w time.Duration, e *error) {
		// If 5 seconds pass, print to stderr, and close waitgroup
		time.Sleep(w)
		*e = fmt.Errorf("Couldn't receive a response")
		wg.Done()
	}(wait, &err)
	go func(e *error) {
		// Read into msg.Packet
		//_, *e = osc.Conn.Read(msg.Packet.Bytes())
		_, *e = osc.Conn.Read(byt)
		wg.Done() // If read is successful, close waitgroup
	}(&err)
	wg.Wait()
	wg.Add(1) // Add to the counter to guard against panicking at a negative counter

	msg.Packet.Write(byt)

	return msg, err
}

func (osc *OSC) Listen(wait time.Duration) (Message, error) {
	msg, err := osc.Receive(wait)
	if err != nil {
		return msg, err
	}

	err = msg.ParseMessage()
	return msg, err
}

func byteToInt32(b []byte) int32 {
	e := binary.BigEndian.Uint32(b[:])
	return int32(e)
}

func byteToFloat32(b []byte) float32 {
	e := binary.BigEndian.Uint32(b[:])
	return math.Float32frombits(e)
}

func allElementsZero(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] != 0 {
			return false
		}
	}
	return true
}
