package osc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"
)

type data []byte

type Message struct {
	Packet   bytes.Buffer
	Addr     string
	Tags     string
	Args     []data
	debugger Debugger
}

func (d *data) Int32() int32 {
	e := binary.BigEndian.Uint32((*d)[:])
	return int32(e)
}

func (d *data) Float32() float32 {
	e := binary.BigEndian.Uint32((*d)[:])
	return math.Float32frombits(e)
}

func (d *data) String() string {
	return string(*d)
}

func float32ToBytes(f float32) []byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], math.Float32bits(f))
	return buf[:]
}

func int32ToBytes(i int32) []byte {
	b := []byte{0, 0, 0, 0}
	binary.BigEndian.PutUint32(b[0:4], uint32(i))
	return b
}

//func zeroBytesToAdd(b []byte) int {
//// The parts of an OSC packet must be divisible by 4 bytes
//return 4 - (len(b) % 4)
//}

func zeroBytesToAdd(l int) int {
	// The parts of an OSC packet must be divisible by 4 bytes
	return 4 - (l % 4)
}

func addZeros(b *[]byte) {
	// The parts of an OSC packet must be divisible by 4 bytes
	//zta := zeroBytesToAdd(*b)
	zta := zeroBytesToAdd(len(*b))
	for i := 0; i < zta; i++ {
		*b = append(*b, byte(0))
	}
}

func (osc *OSC) NewMessage(addr string) *Message {
	msg := &Message{
		Addr:     addr,
		debugger: osc.Debugger,
	}

	return msg
}

func (msg *Message) Debug(m string, args ...any) {
	if msg.debugger != nil {
		msg.debugger.Debug(m, args...)
	}
}

func (msg *Message) MakePacket() error {
	var n int
	var err error
	// Get the address in bytes and pad with zeros
	addrBytes := []byte(msg.Addr)
	addZeros(&addrBytes)
	n, err = msg.Packet.Write(addrBytes)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("could not write AddrBytes")
	}

	// Get the tags in bytes, prefix with comma, and pad with zeros
	tagBytes := append([]byte{','}, []byte(msg.Tags)...)
	addZeros(&tagBytes)
	n, err = msg.Packet.Write(tagBytes)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("could not write TagBytes")
	}

	// The msg.Args is already padded with proper count of zeros
	for _, arg := range msg.Args { // for []data in Args
		n, err = msg.Packet.Write([]byte(arg))
		if err != nil {
			return err
		}
		if n == 0 {
			return errors.New("could not write ArgBytes")
		}
	}

	return nil
}

func (msg *Message) Add(l any) error {
	switch v := l.(type) {
	case int:
		msg.AddInt(int32(v))
	case int32:
		msg.AddInt(v)
	case int64:
		msg.AddInt(int32(v))
	case float32:
		msg.AddFloat(v)
	case float64:
		msg.AddFloat(float32(v))
	case string:
		msg.AddString(v)
	default:
		return errors.New("cannot determine type of argument")
	}
	return nil
}

func (msg *Message) AddString(data string) error {
	// A tag consists of a comma character followed by the tags
	msg.Tags += "s"

	stringBytes := []byte(data)
	addZeros(&stringBytes)

	msg.Args = append(msg.Args, stringBytes)

	return nil
}

func (msg *Message) AddInt(data int32) error {
	// A tag consists of a comma character followed by the tags
	msg.Tags += "i"

	// A int32 should consist of four bytes (32 bits)
	intBytes := int32ToBytes(data)
	addZeros(&intBytes) // This will pad with 4 more zero bytes

	msg.Args = append(msg.Args, intBytes)

	return nil
}

func (msg *Message) AddFloat(data float32) error {
	// A tag consists of a comma character followed by the tags
	msg.Tags += "f"

	// A float32 should consist of four bytes (32 bits)
	floatBytes := float32ToBytes(data)
	addZeros(&floatBytes) // This will pad with 4 more zero bytes

	msg.Args = append(msg.Args, floatBytes)

	return nil
}

func (msg *Message) ParseMessage() error {
	var err error

	// If there is no data in the packet bytes buffer, return err
	if msg.Packet.Len() == 0 {
		err = errors.New("received empty packet")
		return err
	}

	// The OSC Address is the portion before the ','
	//     Write string bytes to msg.Addr until we hit the ','
	msg.Addr, err = msg.Packet.ReadString(',')
	if err != nil {
		return err
	}
	// Trim off the comma we just wrote to msg.Addr
	msg.Addr = strings.TrimSuffix(msg.Addr, ",")
	// Trim off the trailing zeros
	msg.Addr = strings.TrimFunc(msg.Addr, func(r rune) bool {
		return r == 0
	})

	// Tags are single characters indicating the type of data
	//     In the message. 'i': int32, 'f': float32, 's': string
	//     Add the tags until we hit a zero byte
	msg.Tags, err = msg.Packet.ReadString(0)
	// There should be at least one null byte after the tags
	//     to make the tag portion of a length divisible by 4
	//     If already divisible by 4, there will be 4 null bytes
	if err != nil {
		return errors.New("no null byte following tags")
	}

	// Trim off the zero byte we just wrote to msg.Tags
	msg.Tags = strings.TrimSuffix(msg.Tags, string(rune(0)))
	// Unread that zero byte
	msg.Packet.UnreadByte()

	// Inc index over padded zero bytes after the tags
	//   len of tags plus one to account for ','
	msg.Packet.Next(
		zeroBytesToAdd(len(msg.Tags) + 1))

	// If we're out of bounds, exit
	if msg.Packet.Len() == 0 {
		return errors.New("out of bounds")
	}

	// We'll make as many iterations as we have tags
	//
	for tagIndex, tag := range msg.Tags {
		msg.Args = append(msg.Args, []byte{})
		if tag == 's' {
			// Read until hit a zero byte
			msg.Args[tagIndex], err = msg.Packet.ReadBytes(0)
			if err != nil {
				return err
			}
			// Trim off the zero byte just written
			msg.Args[tagIndex] = bytes.TrimSuffix(msg.Args[tagIndex], []byte{0})
			msg.Packet.UnreadByte() // Unread that zero byte
			msg.Packet.Next(
				zeroBytesToAdd(len(msg.Args[tagIndex])))
		}
		if tag == 'i' || tag == 'f' {
			ibuf := make([]byte, 4)
			n, err := msg.Packet.Read(ibuf)
			if err != nil {
				return err
			}
			if n != 4 {
				return errors.New("didn't read all 32 bits")
			}
			msg.Args[tagIndex] = ibuf
		}
		if tag == 'b' {
			err = errors.New(
				"dontains a blob\nNot yet sure how to parse")
			return err
		}
	}

	// If there are still nonzero bytes left
	var byt byte
	for byt, err = msg.Packet.ReadByte(); err != nil; {
		// this err doesn't do anything
		if byt != 0 {
			return errors.New("more data than expected")
		}
	}

	return nil
}
