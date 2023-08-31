package osc

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

func (msg *Message) PrintData() {
	msg.Debug("", "addr", msg.Addr)
	msg.Debug("", "tags", msg.Tags)
	// Iterating a string returns runes???
	for i, tag := range msg.Tags {
		switch tag {
		case 'i':
			fmt.Printf("Data Int: %v\n", msg.Args[i].Int32())
		case 'f':
			fmt.Printf("Data Float: %v\n", msg.Args[i].Float32())
		case 's':
			fmt.Printf("Data String: %v\n", msg.Args[i].String())
		default:
			fmt.Fprintf(os.Stderr,
				"%v\n",
				errors.New("cannot determine type"))
		}
	}
	// write to a buffer and dump it to th log
	var buf bytes.Buffer
	buf.WriteString("Data Bytes: { ")
	for _, arg := range msg.Args {
		for _, b := range arg {
			if b == 0 {
				buf.WriteString("~ ")
				continue
			}
			buf.WriteString(fmt.Sprintf("%v", b))
		}
	}
	buf.WriteString("}\n")
	printPacket(msg.Packet.Bytes())
}

func printPacket(packet []byte) {
	for _, c := range packet {
		if c == 0 {
			fmt.Printf("~")
			continue
		}
		fmt.Printf(" %v ", string(c))
	}
}
