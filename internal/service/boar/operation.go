package boar

import (
	"context"
	"io"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

// ResizeConsoleWindow .
func (svc *Boar) ResizeConsoleWindow(ctx context.Context, id string, height, width uint) (err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return errors.Trace(err)
	}
	return g.ResizeConsoleWindow(ctx, height, width)
}

type executeResult struct {
	output   []byte
	exitCode int
	pid      int
}

// ExecuteGuest .
func (svc *Boar) ExecuteGuest(ctx context.Context, id string, commands []string) (_ *types.ExecuteGuestMessage, err error) {
	defer logErr(err)

	exec := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		output, exitCode, pid, err := g.ExecuteCommand(ctx, commands)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return &executeResult{output: output, exitCode: exitCode, pid: pid}, nil
	}

	res, err := svc.do(ctx, id, miscOp, exec, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}

	er, ok := res.(*executeResult)
	if !ok {
		return nil, errors.Annotatef(errors.ErrInvalidValue, "expect *executeResult but it's %v", res)
	}
	svc.pid2ExitCode.Put(id, er.pid, er.exitCode)
	return &types.ExecuteGuestMessage{
		Pid:      er.pid,
		Data:     er.output,
		ExitCode: er.exitCode,
	}, err
}

// ExecExitCode .
func (svc *Boar) ExecExitCode(id string, pid int) (int, error) {
	exitCode, err := svc.pid2ExitCode.Get(id, pid)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return 0, err
	}
	return exitCode, nil
}

// Cat .
func (svc *Boar) Cat(ctx context.Context, id, path string, dest io.WriteCloser) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		return g.Cat(ctx, path, dest)
	}, nil)
}

// CopyToGuest .
func (svc *Boar) CopyToGuest(ctx context.Context, id, dest string, content chan []byte, override bool) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		return g.CopyToGuest(ctx, dest, content, override)
	}, nil)
}

// Log .
func (svc *Boar) Log(ctx context.Context, id, logPath string, n int, dest io.WriteCloser) (err error) {
	defer logErr(err)

	return svc.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		if g.LambdaOption == nil {
			return g.Log(ctx, n, logPath, dest)
		}

		defer dest.Close()
		_, err := dest.Write(g.LambdaOption.CmdOutput)
		return err
	}, nil)
}
