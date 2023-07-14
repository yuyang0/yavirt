package libvirt

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/utils"
	libvirtgo "libvirt.org/go/libvirt"
)

type ConsoleFlags struct {
	Force    bool
	Safe     bool
	Nonblock bool
}

func (cf *ConsoleFlags) genLibvirtFlags() (flags libvirtgo.DomainConsoleFlags) {
	if cf.Force {
		flags |= libvirtgo.DOMAIN_CONSOLE_FORCE
	}
	if cf.Safe {
		flags |= libvirtgo.DOMAIN_CONSOLE_SAFE
	}
	return
}

func (cf *ConsoleFlags) genStreamFlags() (flags libvirtgo.StreamFlags) {
	if cf.Nonblock {
		flags = libvirtgo.STREAM_NONBLOCK
	}
	return
}

type Console struct {
	Stream *libvirtgo.Stream

	fromQ *utils.BytesQueue
	toQ   *utils.BytesQueue

	quit chan struct{}
	once sync.Once
}

func newConsole(s *libvirtgo.Stream) *Console {
	return &Console{
		Stream: s,
		fromQ:  utils.NewBytesQueue(),
		toQ:    utils.NewBytesQueue(),
		quit:   make(chan struct{}),
	}
}

func (c *Console) needExit(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return true
	case <-c.quit:
		return true
	default:
		return false
	}
}

func (c *Console) From(ctx context.Context, r io.Reader) error {
	buf := make([]byte, 64*1024)
	for {
		if c.needExit(ctx) {
			return nil
		}
		n, err := r.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Errorf("[Console:From] read error: %s", err)
			}
			return err
		}
		if c.needExit(ctx) {
			return nil
		}
		bs := buf[:n]
		// check if input contains ^]
		if bytes.Contains(bs, []byte{uint8(29)}) {
			return nil
		}
		_, err = c.toQ.Write(bytes.Clone(bs))
		if err != nil {
			log.Errorf("[Console:From] write error: %s", err)
			return err
		}
	}
}

func (c *Console) To(ctx context.Context, w io.Writer) error {
	buf := make([]byte, 64*1024)
	for {
		if c.needExit(ctx) {
			return nil
		}
		n, err := c.fromQ.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Errorf("[Console:To] read error: %s", err)
			}
			return err
		}

		if c.needExit(ctx) {
			return nil
		}

		_, err = w.Write(buf[:n])
		if err != nil {
			log.Errorf("[Console:To] write error: %s", err)
			return err
		}
	}
}

func (c *Console) Close() error {
	c.close(true)
	return nil
}

func (c *Console) close(graceful bool) {
	log.Debugf("close called")
	c.once.Do(func() {
		defer func() {
			close(c.quit)
		}()

		// c.Stream.EventRemoveCallback() //nolint
		var err error
		if graceful {
			err = c.Stream.Finish()
		} else {
			err = c.Stream.Abort()
		}
		if err != nil {
			log.Errorf("Can't finish or abort stream: %v", err)
		}
		c.Stream.Free() //nolint
		c.fromQ.Close()
		c.toQ.Close()
	})
}

func sendAll(stream *libvirtgo.Stream, bs []byte) error {
	for len(bs) > 0 {
		n, err := stream.Send(bs)
		if err != nil {
			return err
		}
		bs = bs[n:]
	}
	return nil
}

// For block stream IO
func (c *Console) AddReadWriter() error {
	ctx := context.Background()
	go func() {
		defer log.Infof("[AddReadWriter] Send goroutine exit")
		for {
			if c.needExit(ctx) {
				return
			}
			bs, err := c.toQ.Pop()
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when write to console toQ queue: %s", err)
				return
			}
			if c.needExit(ctx) {
				return
			}
			err = sendAll(c.Stream, bs)
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when write to console stream: %s", err)
				return
			}
			log.Debugf("[AddReadWriter] sent to stream: %v\r\n", bs)
		}
	}()
	go func() {
		defer log.Infof("[AddReadWriter] Recv goroutine exit")
		buf := make([]byte, 100*1024)
		for {
			if c.needExit(ctx) {
				return
			}
			n, err := c.Stream.Recv(buf)
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when read from console stream: %s", err)
				return
			}
			bs := buf[:n]
			log.Debugf("[AddReadWriter] recv from stream: %v\r\n", bs)

			if c.needExit(ctx) {
				return
			}
			_, err = c.fromQ.Write(bs)
			if err != nil {
				log.Warnf("[AddReadWriter] Got error when write to console queue: %s", err)
				return
			}
		}
	}()
	return nil
}

// For nonblock stream IO
func (c *Console) AddCallback() error {
	cb := func(stream *libvirtgo.Stream, evt libvirtgo.StreamEventType) {
		var err error
		var n int

		defer func() {
			if err != nil {
				log.Debugf("[Event.Callback] exit with error: %s", err)
			}
		}()
		switch {
		case evt&libvirtgo.STREAM_EVENT_READABLE > 0:
			buf := make([]byte, 100*1024)
			n, err = c.Stream.Recv(buf)
			log.Debugf("[Event Callback] read: %d, %s, %s", n, err, string(buf[:n]))
			if n == -2 { // EAGAIN
				log.Debugf("[Event.Callback] readable EAGAIN")
				return
			}
			if err != nil {
				c.close(n == 0)
				return
			}

			bs := bytes.Clone(buf[:n])
			_, err = c.fromQ.Write(bs)
			if err != nil {
				c.close(n == 0)
				return
			}
		case evt&libvirtgo.STREAM_EVENT_WRITABLE > 0:
			log.Debugf("[Event callback] write ready")
			_, err := c.toQ.HandleHead(func(p []byte) (int, error) {
				return c.Stream.Send(p)
			})
			if err != nil {
				c.close(false)
			}
		case evt&(libvirtgo.STREAM_EVENT_ERROR|libvirtgo.STREAM_EVENT_HANGUP) > 0:
			c.close(false)
		}
	}

	err := c.Stream.EventAddCallback(libvirtgo.STREAM_EVENT_READABLE, cb)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}
