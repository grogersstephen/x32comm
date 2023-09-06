package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"time"

	"log/slog"

	"github.com/grogersstephen/x32comm/osc"
	"github.com/urfave/cli/v2"
)

const (
	// UPDATE: X32 faders apparently only have 10bit resolution
	FADER_RESOLUTION float32 = 1 << 10 // 10bit
	//FADER_RESOLUTION float32 = 256 // 8bit
)

var x *x32

var logLevel = &slog.LevelVar{}

type x32 struct {
	osc.OSC
	ctx context.Context
}

var beforeFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "port",
		Aliases: []string{"p"},
		Value:   "10024",
	},
	&cli.StringFlag{
		Name:  "dest",
		Usage: "destination address",
		Value: "45.56.112.149:10023",
	},
}

func joinFlags(args ...[]cli.Flag) []cli.Flag {
	out := []cli.Flag{}
	for _, flags := range args {
		out = append(out, flags...)
	}
	return out
}

func main() {
	logLevel.Set(slog.LevelDebug)

	slogOpts := &slog.HandlerOptions{
		Level: logLevel,
	}

	logr := slog.New(slog.NewTextHandler(os.Stdout, slogOpts))

	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	defer close(sig)

	go func() {
		defer cancel()
		signal.Notify(sig, os.Interrupt)
		<-sig
		logr.Info("close")
	}()

	before := func(c *cli.Context) (err error) {
		x = &x32{
			OSC: osc.OSC{
				Debugger: logr.With("entity", "osc"),
			},
			ctx: c.Context,
		}

		dest := c.String("dest")

		x.Destination, err = net.ResolveUDPAddr("udp", dest)
		if err != nil {
			return fmt.Errorf("resolve dest udp addr: %v", err)
		}

		logr.Info(x.Destination.String())

		port := c.String("port")

		x.Client, err = net.ResolveUDPAddr("udp", ":"+port)
		if err != nil {
			return fmt.Errorf("resolve client udp addr: %v", err)
		}

		err = x.Dial()
		if err != nil {
			return fmt.Errorf("dial: %v", err)
		}

		go func() {
			// read from channel and close conn; this channel should be the ctx canceled from signals channel
			for range c.Done() {
				fmt.Println("done here")
				x.Conn.Close()
			}
		}()

		return nil
	}

	var message string
	var floatArg float64
	var channel, fadeStart, fadeStop int
	var fadeDuration time.Duration

	app := &cli.App{
		Name:  "x32 comm",
		Usage: "communicates via osc with x32",
		Before: func(ctx *cli.Context) error {

			err := before(ctx)
			if err != nil {
				fmt.Println("before err", err)
				return err
			}
			return nil
		},
		Flags: beforeFlags,
		Commands: []*cli.Command{
			{
				Name:  "set",
				Usage: "Send a float message expecting no response",
				Flags: joinFlags(beforeFlags, []cli.Flag{
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
				}),
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
						return errors.New("please select a channel 1-32")
					}
					if levelS == "" {
						return errors.New("please select a level 0-100")
					}
					channelI, err := strconv.Atoi(channelS)
					if err != nil {
						return errors.New("please select a channel 1-32")
					}
					levelI, err := strconv.ParseFloat(levelS, 32)
					if err != nil {
						return errors.New("please select a level 0-100")
					}
					levelF := float32(levelI) / float32(100)
					err = x.setChFader(channelI, levelF)
					return err
				},
			},
			{
				Name:  "get",
				Usage: "Send a message and await a response",
				Flags: joinFlags(beforeFlags, []cli.Flag{
					&cli.StringFlag{
						Name:        "message",
						Aliases:     []string{"m"},
						Destination: &message,
						Required:    true,
					},
				}),
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
				Flags: joinFlags(beforeFlags),
				Action: func(cCtx *cli.Context) error {
					channelS := cCtx.Args().Get(0)
					channelI, err := strconv.Atoi(channelS)
					if err != nil {
						return err
					}
					fader, err := x.getChFader(channelI)
					if err != nil {
						return fmt.Errorf("get ch fader: %v", err)
					}
					fmt.Printf("%v", fader)
					fmt.Fprint(os.Stderr, "\n")
					return nil
				},
			},
			{
				Name:  "listen",
				Usage: "listen for a message",
				Flags: joinFlags(beforeFlags),
				Action: func(cCtx *cli.Context) error {
					out, err := x.listen(cCtx.Context, 9*time.Second)
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
				Flags: joinFlags(beforeFlags, []cli.Flag{
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
				}),
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
				Flags: joinFlags(beforeFlags, []cli.Flag{
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
				}),
				Action: func(cCtx *cli.Context) error {
					fadeStartF := float32(fadeStart) / float32(100)
					fadeStopF := float32(fadeStop) / float32(100)
					x.makeFade(channel, fadeStartF, fadeStopF, fadeDuration)
					return nil
				},
			},
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}

}

func (x *x32) listen(ctx context.Context, wait time.Duration) (any, error) {
	msg, err := x.Listen(ctx, wait)
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
	return nil, errors.New("cannot determine data type")
}

func (x *x32) sendAndListen(message string, wait time.Duration) (any, error) {
	err := x.SendString(message)
	if err != nil {
		return "", fmt.Errorf("send string: %w", err)
	}
	out, err := x.listen(x.ctx, wait)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	return out, err
}

func (x *x32) getChFader(ch int) (float32, error) {
	msg := getFaderPath(ch)

	out, err := x.sendAndListen(msg, time.Second*9)
	if err != nil {
		return 0, fmt.Errorf("send and listen: %w", err)
	}
	fader, ok := out.(float32)
	if !ok {
		return 0, errors.New("not a float")
	}
	return fader, nil
}

func (x *x32) setChFader(ch int, level float32) error {

	msg := getFaderPath(ch)
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
	err = x.Send(msg)
	return err
}

func getFaderPath(ch int) string {

	return filepath.Join("/ch", fmt.Sprintf("%02d", ch), "mix/fader")
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
