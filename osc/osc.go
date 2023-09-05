package osc

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/grogersstephen/x32comm/internal/conman"
	"golang.org/x/sync/errgroup"
)

type Debugger interface {
	Debug(msg string, args ...any)
}

type OSC struct {
	Destination *net.UDPAddr
	Client      *net.UDPAddr
	Conn        io.ReadWriteCloser
	Debugger    Debugger
}

func (osc *OSC) Debug(msg string, args ...any) {
	if osc.Debugger != nil {
		osc.Debugger.Debug(msg, args...)
	}
}

func (osc *OSC) Dial() error {
	conn, err := net.DialUDP("udp", osc.Client, osc.Destination)
	osc.Debug("osc.Destination", "host", osc.Destination.IP, "port", osc.Destination.Port)
	osc.Debug("osc.Conn", "local", conn.LocalAddr().String())

	if err != nil {
		return err
	}

	tmp := os.TempDir()
	ts := time.Now()
	fp := filepath.Join(tmp, fmt.Sprintf("x32_conman_%d.txt", ts.Unix()))
	if err := os.Mkdir(fp, 0750); err != nil {
		return fmt.Errorf("mkdir; %s: %w", fp, err)
	}

	cm, err := conman.NewConman(conman.WithRef(conn), conman.WithStructuredDrains(fp))
	if err != nil {
		return fmt.Errorf("new conman: %w", err)
	}

	osc.Debug("copied reads and writes", "fp", fp)

	osc.Conn = cm
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
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return err
		}
	}
	return nil
}

func (osc *OSC) Send(msg interface {
	MakePacket() ([]byte, error)
}) error {
	b, err := msg.MakePacket()
	if err != nil {
		return err
	}

	// Sends a message using the Conn from net.DialUDP
	_, err = osc.Conn.Write(b)
	return err
}

func (osc *OSC) Receive(ctx context.Context, wait time.Duration) (Message, error) {
	// Waits for and reads a message from Conn from net.DialUDP
	// wait is wait time in milliseconds
	var (
		msg Message
	)

	gctx, done := context.WithTimeout(ctx, wait)

	// the ctx here will be use for later
	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(gctx)
	defer done()
	_ = ctx

	byt := make([]byte, 2048)

	osc.Debug("Waiting for Reply...")

	// pw, pr := io.Pipe()

	eg.Go(func() error {
		// Read into msg.Packet
		fmt.Println("before read")
		if _, err := osc.Conn.Read(byt); err != nil {
			if err != io.EOF {
				return fmt.Errorf("read: %w", err)
			}
		}
		fmt.Println("after read")
		return nil
	})

	if _, err := msg.Packet.Write(byt); err != nil {
		if err != io.EOF {
			return msg, fmt.Errorf("write: %w", err)
		}
	}
	if err := eg.Wait(); err != nil {
		return msg, fmt.Errorf("wait: %v", err)
	}
	fmt.Println("byt", string(byt))
	return msg, nil
}

func (osc *OSC) Listen(ctx context.Context, wait time.Duration) (Message, error) {

	msg, err := osc.Receive(ctx, wait)
	if err != nil {
		return msg, fmt.Errorf("receive: %w", err)
	}

	err = msg.ParseMessage()
	if err != nil {
		return msg, fmt.Errorf("parse msg: %w", err)
	}
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
