package boar

import (
	"context"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

// ControlGuest .
func (svc *Boar) ControlGuest(ctx context.Context, id, operation string, force bool) (err error) {
	switch operation {
	case types.OpStart:
		err = svc.startGuest(ctx, id)
	case types.OpStop:
		err = svc.stopGuest(ctx, id, force)
	case types.OpDestroy:
		_, err = svc.destroyGuest(ctx, id, force)
	case types.OpSuspend:
		err = svc.suspendGuest(ctx, id)
	case types.OpResume:
		err = svc.resumeGuest(ctx, id)
	}

	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return errors.Trace(err)
	}

	return nil
}

// destroyGuest destroys a guest.
func (svc *Boar) destroyGuest(ctx context.Context, id string, force bool) (<-chan error, error) {
	var done <-chan error
	err := svc.ctrl(ctx, id, destroyOp, func(g *guest.Guest) (de error) {
		done, de = g.Destroy(ctx, force)
		return
	}, nil)
	return done, err
}

// stopGuest stops a guest.
func (svc *Boar) stopGuest(ctx context.Context, id string, force bool) error {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if err := g.Stop(ctx, force); err != nil {
			return nil, errors.Trace(err)
		}

		return nil, nil //nolint
	}
	_, err := svc.do(ctx, id, shutOp, do, nil)
	return err
}

// startGuest boots a guest.
func (svc *Boar) startGuest(ctx context.Context, id string) error {
	do := func(ctx context.Context) (any, error) {
		g, err := svc.loadGuest(ctx, id)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if err := g.Start(ctx); err != nil {
			return nil, errors.Trace(err)
		}

		if g.LambdaOption != nil && !g.LambdaStdin {
			output, exitCode, pid, err := g.ExecuteCommand(ctx, g.LambdaOption.Cmd)
			if err != nil {
				return nil, errors.Trace(err)
			}
			g.LambdaOption.CmdOutput = output
			g.LambdaOption.ExitCode = exitCode
			g.LambdaOption.Pid = pid

			if err = g.Save(); err != nil {
				return nil, errors.Trace(err)
			}
		}
		return nil, nil //nolint
	}
	defer log.Debugf("exit manager.Start")
	_, err := svc.do(ctx, id, bootOp, do, nil)
	return err
}

// suspendGuest suspends a guest.
func (svc *Boar) suspendGuest(ctx context.Context, id string) error {
	return svc.ctrl(ctx, id, bootOp, func(g *guest.Guest) error {
		return g.Suspend()
	}, nil)
}

// resumeGuest resumes a suspended guest.
func (svc *Boar) resumeGuest(ctx context.Context, id string) error {
	return svc.ctrl(ctx, id, bootOp, func(g *guest.Guest) error {
		return g.Resume()
	}, nil)
}
