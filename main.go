package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/grogersstephen/x32comm/osc"
	"github.com/urfave/cli/v2"
)

const (
	// UPDATE: X32 faders apparently only have 10bit resolution
	FADER_RESOLUTION float32 = 1024 // 10bit
	//FADER_RESOLUTION float32 = 256 // 8bit
)

type x32 struct {
	osc.OSC
}

func main() {
	var err error
	var x x32

	x.Destination, err = net.ResolveUDPAddr("udp", "45.56.112.149:10023")
	if err != nil {
		log.Fatal(err)
	}
	// Cannot use localhost or 127.0.0.1
	x.Client, err = net.ResolveUDPAddr("udp", ":10024")
	//	x.Client, err = net.ResolveUDPAddr("udp", "127.0.0.1:10024")
	//x.Client, err = net.ResolveUDPAddr("udp", "192.168.0.173:10024")
	//x.Client, err = net.ResolveUDPAddr("udp", "192.168.213.55:10024")
	if err != nil {
		log.Fatal(err)
	}

	err = x.Dial()
	if err != nil {
		log.Fatal(err)
	}

	defer x.Conn.Close()

	var message string
	var floatArg float64
	var channel, fadeStart, fadeStop int
	var fadeDuration time.Duration
	//var stringArg string

	app := &cli.App{
		Name:  "x32 comm",
		Usage: "communicates via osc with x32",
		Commands: []*cli.Command{
			{
				Name:  "set",
				Usage: "Send a float message expecting no response",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "message",
						Aliases:     []string{"m"},
						Destination: &message,
						Required:    true,
					},
					&cli.Float64Flag{
						Name:        "float",
						Aliases:     []string{"f"},
						Destination: &floatArg,
						Required:    false,
					},
				},
				Action: func(cCtx *cli.Context) error {
					err := x.compose(message, floatArg)
					return err
				},
			},
			{
				Name:  "setChFader",
				Usage: "Set a channel fader to a certain level",
				Action: func(cCtx *cli.Context) error {
					channelS := cCtx.Args().Get(0)
					levelS := cCtx.Args().Get(1)
					if channelS == "" {
						return fmt.Errorf("please select a channel 1-32")
					}
					if levelS == "" {
						return fmt.Errorf("please select a level 0-100")
					}
					channelI, err := strconv.Atoi(channelS)
					if err != nil {
						return fmt.Errorf("please select a channel 1-32")
					}
					levelI, err := strconv.ParseFloat(levelS, 32)
					if err != nil {
						return fmt.Errorf("please select a level 0-100")
					}
					levelF := float32(levelI) / float32(100)
					err = x.setChFader(channelI, levelF)
					return err
				},
			},
			{
				Name:  "get",
				Usage: "Send a message and await a response",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "message",
						Aliases:     []string{"m"},
						Destination: &message,
						Required:    true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					out, err := x.sendAndListen(message, 9*time.Second)
					if err != nil {
						return err
					}
					fmt.Printf("Type: %T\n", out)
					fmt.Printf("Value: %v\n", out)
					return nil
				},
			},
			{
				Name:  "getChFader",
				Usage: "Get a channel fader level",
				Action: func(cCtx *cli.Context) error {
					channelS := cCtx.Args().Get(0)
					channelI, err := strconv.Atoi(channelS)
					if err != nil {
						return err
					}
					fader, err := x.getChFader(channelI)
					if err != nil {
						return err
					}
					fmt.Printf("%v", fader)
					fmt.Fprint(os.Stderr, "\n")
					return nil
				},
			},
			{
				Name:  "listen",
				Usage: "listen for a message",
				Action: func(cCtx *cli.Context) error {
					out, err := x.listen(9 * time.Second)
					if err != nil {
						return err
					}
					fmt.Printf("Type: %T\n", out)
					fmt.Printf("Value: %v\n", out)
					return nil
				},
			},
			{
				Name:  "fadeTo",
				Usage: "fade a channel to a specified level from its current level ",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:        "ch",
						Aliases:     []string{"c"},
						Usage:       "select a channel to operate on",
						Destination: &channel,
						Required:    true,
					},
					&cli.IntFlag{
						Name:        "stop",
						Aliases:     []string{"b"},
						Usage:       "define a stop pos for fade from 0 - 100",
						Destination: &fadeStop,
						Required:    true,
					},
					&cli.DurationFlag{
						Name:        "duration",
						Aliases:     []string{"d"},
						Usage:       "define the desired duration over which to make the fade",
						Value:       2 * time.Second,
						Destination: &fadeDuration,
					},
				},
				Action: func(cCtx *cli.Context) error {
					currentLevel, err := x.getChFader(channel)
					fmt.Printf("current level: %v\n", currentLevel)
					if err != nil {
						return err
					}
					fadeStopF := float32(fadeStop) / float32(100)
					fmt.Printf("fade stop: %v\n", fadeStopF)
					fmt.Printf("fade duration: %v\n", fadeDuration)
					x.makeFade(channel, currentLevel, fadeStopF, fadeDuration)
					return nil
				},
			},
			{
				Name:  "fade",
				Usage: "fade a channel from one level to another",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:        "ch",
						Aliases:     []string{"c"},
						Usage:       "select a channel to operate on",
						Destination: &channel,
						Required:    true,
					},
					&cli.IntFlag{
						Name:        "start",
						Aliases:     []string{"a"},
						Usage:       "define a start pos for fade from 0 - 100",
						Destination: &fadeStart,
						Required:    true,
					},
					&cli.IntFlag{
						Name:        "stop",
						Aliases:     []string{"b"},
						Usage:       "define a stop pos for fade from 0 - 100",
						Destination: &fadeStop,
						Required:    true,
					},
					&cli.DurationFlag{
						Name:        "duration",
						Aliases:     []string{"d"},
						Usage:       "define the desired duration over which to make the fade",
						Value:       2 * time.Second,
						Destination: &fadeDuration,
					},
				},
				Action: func(cCtx *cli.Context) error {
					fadeStartF := float32(fadeStart) / float32(100)
					fadeStopF := float32(fadeStop) / float32(100)
					x.makeFade(channel, fadeStartF, fadeStopF, fadeDuration)
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func (x *x32) listen(wait time.Duration) (any, error) {
	msg, err := x.Listen(wait)
	if err != nil {
		return nil, err
	}

	switch msg.Tags[0] {
	case 's':
		return msg.Args[0].String(), nil
	case 'i':
		return msg.Args[0].Int32(), nil
	case 'f':
		return msg.Args[0].Float32(), nil
	}
	return nil, fmt.Errorf("cannot determine data type:")
}

func (x *x32) sendAndListen(message string, wait time.Duration) (any, error) {
	err := x.SendString(message)
	if err != nil {
		return "", err
	}
	out, err := x.listen(wait)
	return out, err
}

func (x *x32) getChFader(ch int) (float32, error) {
	chS := fmt.Sprintf("%d", ch)
	if ch < 10 {
		chS = "0" + chS
	}
	msg := filepath.Join("/ch/", chS, "/mix/fader")
	out, err := x.sendAndListen(msg, time.Second*9)
	if err != nil {
		return 0, err
	}
	fader, ok := out.(float32)
	if ok != true {
		return 0, fmt.Errorf("not a float")
	}
	return fader, nil
}

func (x *x32) setChFader(ch int, level float32) error {
	chS := fmt.Sprintf("%d", ch)
	if ch < 10 {
		chS = "0" + chS
	}
	msg := filepath.Join("/ch/", chS, "/mix/fader")
	err := x.compose(msg, level)
	return err
}

func (x *x32) compose(message string, arg ...any) error {
	// Only allows for one argument
	msg := x.NewMessage(message)
	err := msg.Add(arg[0])
	if err != nil {
		return err
	}
	err = x.Send(*msg)
	return err
}

func getFaderPath(ch int) string {
	var path, zerodigit, chS string
	chS = fmt.Sprintf("%d", ch)
	if len(chS) == 1 {
		zerodigit = "0"
	}
	path = filepath.Join("/ch", zerodigit+chS, "mix/fader")
	return path
}

func getDist(x, y int) int {
	// Returns the absolute distance between two integers
	if x < y {
		return y - x
	}
	return x - y
}

func (x *x32) makeFade(ch int, start, stop float32, d time.Duration) {
	path := getFaderPath(ch) // get the fader path of the osc command given the channel number
	// Determine if we'll be increasing or decreasing as we iterate from start to stop
	step := 1
	if start > stop {
		step = -1
	}
	// Initialize startI and stopI ints on a scale from 0 to FADER_RESOLUTION
	startI := int(start * FADER_RESOLUTION)
	stopI := int(stop * FADER_RESOLUTION)
	dist := getDist(startI, stopI) // absolute distance between start and stop
	// delay is:    desired duration / distance
	delayF := float64(d.Milliseconds()) / float64(dist)
	delay, _ := time.ParseDuration(fmt.Sprintf("%vms", delayF))
	margin := int(.01 * float64(dist)) // margins are with 2% of the target
	// Iterate from start, with a step of 1 or -1, until we reach the desired margin of stop
	for i := startI; (stopI-i) > margin || (stopI-i) < -margin; i += step {
		v := float32(i) / FADER_RESOLUTION // convert to a float32 on a scale from 0 to 1
		// Create our message
		// append the float32 to the message
		// send the message
		x.compose(path, v)
		time.Sleep(delay) // sleep the calculated delay
	}
}
