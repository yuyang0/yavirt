package util

import (
	"context"

	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
)

type ctxKey string

const calicoHandlerKey ctxKey = "CalicoHandler"

// append calico handler to context .
func SetCalicoHandler(ctx context.Context, caliHandler *calihandler.Handler) context.Context {
	return context.WithValue(ctx, calicoHandlerKey, caliHandler)
}

// CalicoHandler .
func GetCalicoHandler(ctx context.Context) (*calihandler.Handler, error) {
	switch hand, ok := ctx.Value(calicoHandlerKey).(*calihandler.Handler); {
	case !ok:
		fallthrough
	case hand == nil:
		return nil, errors.Annotatef(errors.ErrInvalidValue, "nil *calihandler.Handler")

	default:
		return hand, nil
	}
}
