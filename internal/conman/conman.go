package conman

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var _ io.ReadWriteCloser = &ConMan{}

type Debugger interface {
	Debug(msg string, args ...any)
}
type ConMan struct {
	mux       sync.RWMutex
	ref       io.ReadWriteCloser
	drainPath io.ReadWriteCloser
	// debugger  Debugger
}

func NewConman(opts ...Option) (*ConMan, error) {
	cm := &ConMan{}
	for _, opt := range opts {
		if err := opt.apply(cm); err != nil {
			return nil, err
		}
	}

	if cm.drainPath == nil || cm.ref == nil {
		return nil, errors.New("empty stuff")
	}

	return cm, nil
}

func (c *ConMan) Read(in []byte) (int, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	tee := io.TeeReader(c.ref, c.drainPath)
	return tee.Read(in)
}

func (c *ConMan) Write(in []byte) (int, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	nw := io.MultiWriter(c.ref, c.drainPath)
	return nw.Write(in)
}

func (c *ConMan) Close() (err error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if terr := c.drain(); terr != nil {
		err = errors.Join(err, fmt.Errorf("drain: %w", terr))
	}

	if terr := c.ref.Close(); terr != nil {
		err = errors.Join(err, fmt.Errorf("close: %w", terr))
	}

	return err
}

func (c *ConMan) drain() (err error) {
	if terr := c.drainPath.Close(); terr != nil {
		err = errors.Join(err, fmt.Errorf("close: %w", terr))
	}

	return err
}
