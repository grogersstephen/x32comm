package osc

import (
	"fmt"
	"os"
)

func (msg *Message) PrintData() {
	fmt.Printf("Addr: %s\n", msg.Addr)
	fmt.Printf("Tags: %s\n", msg.Tags)
	// Iterating a string returns runes???
	for i, tag := range msg.Tags {
		switch tag {
		case 'i':
			fmt.Printf("Data Int: %v\n",
				msg.Args[i].Int32())
		case 'f':
			fmt.Printf("Data Float: %v\n",
				msg.Args[i].Float32())
		case 's':
			fmt.Printf("Data String: %v\n",
				msg.Args[i].String())
		default:
			fmt.Fprintf(os.Stderr,
				"%v\n",
				fmt.Errorf("Cannot Determine Type"))
		}
	}
	fmt.Printf("Data Bytes: { ")
	for _, arg := range msg.Args {
		for _, b := range arg {
			if b == 0 {
				fmt.Fprintf(os.Stdout, "~ ")
				continue
			}
			fmt.Fprintf(os.Stdout, "%v ", b)
		}
	}
	fmt.Printf("}\n")
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
