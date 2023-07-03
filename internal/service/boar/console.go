package boar

import (
	"context"
	"io"

	"github.com/projecteru2/yavirt/internal/meta"
	virtypes "github.com/projecteru2/yavirt/internal/virt/types"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// AttachGuest .
func (svc *Boar) AttachGuest(ctx context.Context, id string, stream io.ReadWriteCloser, flags virtypes.OpenConsoleFlags) (err error) {
	defer logErr(err)

	g, err := svc.loadGuest(ctx, id)
	if err != nil {
		return errors.Trace(err)
	}

	if g.LambdaOption != nil {
		if err = g.Wait(meta.StatusRunning, false); err != nil {
			return errors.Trace(err)
		}
		flags.Commands = g.LambdaOption.Cmd
	}

	return g.AttachConsole(ctx, stream, flags)
}
