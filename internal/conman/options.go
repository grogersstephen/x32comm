package conman

import (
	"encoding/hex"
	"io"
	"log/slog"
	"os"
)

type Option interface {
	apply(*ConMan) error
}

type applyOptionFunc func(c *ConMan) error

func (f applyOptionFunc) apply(c *ConMan) error {
	return f(c)
}
func WithRef(ref io.ReadWriteCloser) Option {
	return applyOptionFunc(func(c *ConMan) error {

		c.ref = ref
		return nil
	})
}

func WithDrainPath(drainPath string) Option {
	return applyOptionFunc(func(c *ConMan) error {

		f, err := os.Open(drainPath)
		if err != nil {
			return err
		}
		c.drainPath = f
		return nil
	})
}

func WithStructuredDrains(drainPath string) Option {
	return applyOptionFunc(func(c *ConMan) error {

		f, err := os.Create(drainPath)
		if err != nil {
			return err
		}
		handler := slog.NewJSONHandler(f, nil)

		logger := slog.New(handler)

		c.drainPath = &StructuredWriter{Logger: logger, f: f}
		return nil
	})
}

type StructuredWriter struct {
	*slog.Logger
	f io.ReadWriteCloser
}

func (sw *StructuredWriter) Write(in []byte) (int, error) {
	sw.Info("write", "hex", hex.EncodeToString(in), "bytes", in, "string", string(in))

	return len(in), nil
}

func (sw *StructuredWriter) Read(in []byte) (int, error) {
	sw.Info("read", "hex", hex.EncodeToString(in), "bytes", in, "string", string(in))

	return sw.f.Read(in)
}

func (sw *StructuredWriter) Close() error {
	return sw.f.Close()
}
