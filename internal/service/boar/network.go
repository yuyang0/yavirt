package boar

import (
	"context"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/virt/guest"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"

	vlanhandler "github.com/projecteru2/yavirt/internal/vnet/handler/vlan"
)

// ConnectNetwork .
func (svc *Boar) ConnectNetwork(ctx context.Context, id, network, ipv4 string) (cidr string, err error) {
	var ip meta.IP

	if err := svc.ctrl(ctx, id, miscOp, func(g *guest.Guest) (ce error) {
		ip, ce = g.ConnectExtraNetwork(network, ipv4)
		return ce
	}, nil); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
		return "", errors.Trace(err)
	}

	return ip.CIDR(), nil
}

// DisconnectNetwork .
func (svc *Boar) DisconnectNetwork(ctx context.Context, id, network string) (err error) {
	err = svc.ctrl(ctx, id, miscOp, func(g *guest.Guest) error {
		return g.DisconnectExtraNetwork(network)
	}, nil)
	if err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// NetworkList .
func (svc *Boar) NetworkList(ctx context.Context, drivers []string) ([]*types.Network, error) {
	drv := map[string]struct{}{}
	for _, driver := range drivers {
		drv[driver] = struct{}{}
	}

	networks := []*types.Network{}
	switch svc.Host.NetworkMode {
	case vnet.NetworkCalico:
		if _, ok := drv[vnet.NetworkCalico]; svc.caliHandler == nil || !ok {
			break
		}
		for _, poolName := range svc.caliHandler.PoolNames() {
			subnet, err := svc.caliHandler.GetIPPoolCidr(ctx, poolName)
			if err != nil {
				log.ErrorStack(err)
				metrics.IncrError()
				return nil, err
			}

			networks = append(networks, &types.Network{
				Name:    poolName,
				Subnets: []string{subnet},
			})
		}
		return networks, nil
	case vnet.NetworkVlan: // vlan
		if _, ok := drv[vnet.NetworkVlan]; !ok {
			break
		}
		handler := vlanhandler.New("", svc.Host.Subnet)
		networks = append(networks, &types.Network{
			Name:    "vlan",
			Subnets: []string{handler.GetCidr()},
		})
	}

	return networks, nil
}
