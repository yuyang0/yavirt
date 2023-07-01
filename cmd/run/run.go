package run

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/util"
	"github.com/projecteru2/yavirt/internal/virt/guest/manager"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/calico"
	calinet "github.com/projecteru2/yavirt/internal/vnet/calico"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/idgen"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/projecteru2/yavirt/pkg/store"
)

var runtime Runtime

// Runner .
type Runner func(*cli.Context, Runtime) error

// Runtime .
type Runtime struct {
	Ctx           context.Context
	CancelFn      context.CancelFunc
	Host          *models.Host
	Device        *device.Driver
	CalicoDriver  *calinet.Driver
	CalicoHandler *calihandler.Handler
	Guest         manager.Manager
}

// VirtContext .
func (r Runtime) VirtContext() context.Context {
	return util.SetCalicoHandler(r.Ctx, r.CalicoHandler)
}

// ConvDecimal .
func (r Runtime) ConvDecimal(ipv4 string) int64 {
	if len(ipv4) < 1 {
		return 0
	}

	var dec, err = netx.IPv4ToInt(ipv4)
	if err != nil {
		panic(err)
	}

	return dec
}

// Run .
func Run(fn Runner) cli.ActionFunc {
	return func(c *cli.Context) error {
		cfg := &configs.Conf

		if err := cfg.Load([]string{c.String("config")}); err != nil {
			return errors.Trace(err)
		}
		if err := cfg.Prepare(c); err != nil {
			return err
		}
		runtime.Ctx, runtime.CancelFn = context.WithTimeout(context.Background(), time.Duration(c.Int("timeout"))*time.Second)
		runtime.Guest = manager.New()
		if err := setup(); err != nil {
			return errors.Trace(err)
		}

		return fn(c, runtime)
	}
}

func setup() error {
	// always send log to stdout
	if _, err := log.Setup(configs.Conf.LogLevel, "", configs.Conf.LogSentry); err != nil {
		return err
	}
	if err := store.Setup(configs.Conf, nil); err != nil {
		return errors.Trace(err)
	}

	if err := setupHost(); err != nil {
		return errors.Trace(err)
	}

	idgen.Setup(runtime.Host.ID, time.Now())

	if runtime.Host.NetworkMode == vnet.NetworkCalico {
		if err := setupCalico(); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func setupHost() (err error) {
	if runtime.Host, err = models.LoadHost(); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func setupCalico() (err error) {
	if endps := os.Getenv("ETCD_ENDPOINTS"); len(endps) < 1 {
		if err = os.Setenv("ETCD_ENDPOINTS", strings.Join(configs.Conf.Etcd.Endpoints, ",")); err != nil {
			return
		}
	}

	if runtime.Device, err = device.New(); err != nil {
		return
	}

	if runtime.CalicoDriver, err = calico.NewDriver(configs.Conf.Calico.ConfigFile, configs.Conf.Calico.PoolNames); err != nil {
		return
	}

	var outboundIP string
	if outboundIP, err = netx.GetOutboundIP(configs.Conf.Core.Addrs[0]); err != nil {
		return
	}

	runtime.CalicoHandler = calihandler.New(runtime.Device, runtime.CalicoDriver, configs.Conf.Calico.PoolNames, outboundIP)
	// err = runtime.CalicoHandler.InitGateway(configs.Conf.Calico.GatewayName)

	return
}
