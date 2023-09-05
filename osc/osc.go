package osc

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

type Debugger interface {
	Debug(msg string, args ...any)
}

type OSC struct {
	Destination *net.UDPAddr
	Client      *net.UDPAddr
	Conn        *net.UDPConn
	Debugger    Debugger
}

func (osc *OSC) Debug(msg string, args ...any) {
	if osc.Debugger != nil {
		osc.Debugger.Debug(msg, args...)
	}
}

func (osc *OSC) Dial() (err error) {
	osc.Conn, err = net.DialUDP("udp", osc.Client, osc.Destination)
	osc.Debug("osc.Destination", "host", osc.Destination.IP, "port", osc.Destination.Port)
	osc.Debug("osc.Conn", "local", osc.Conn.LocalAddr().String())

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

func (osc *OSC) Receive(ctx context.Context, wait time.Duration) (Message, error) {
	// Waits for and reads a message from Conn from net.DialUDP
	// wait is wait time in milliseconds
	var (
		msg Message
	)

	ctx, done := context.WithTimeout(ctx, wait)

	// the ctx here will be use for later
	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(ctx)
	defer done()
	_ = ctx

	byt := make([]byte, 8192)

	osc.Debug("Waiting for Reply...")

	// pw, pr := io.Pipe()

	eg.Go(func() error {
		// Read into msg.Packet
		if _, err := osc.Conn.Read(byt); err != nil {
			return err
		}
		return nil
	})

	if _, err := msg.Packet.Write(byt); err != nil {
		return msg, err
	}

	return msg, eg.Wait()
}

func (osc *OSC) Listen(ctx context.Context, wait time.Duration) (Message, error) {

	msg, err := osc.Receive(ctx, wait)
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
