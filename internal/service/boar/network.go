package boar

import (
	"context"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/pkg/log"

	vlanhandler "github.com/projecteru2/yavirt/internal/vnet/handler/vlan"
)

// ConnectNetwork .
func (svc *Boar) ConnectNetwork(ctx context.Context, id, network, ipv4 string) (cidr string, err error) {
	if cidr, err = svc.guest.ConnectExtraNetwork(ctx, id, network, ipv4); err != nil {
		log.ErrorStack(err)
		metrics.IncrError()
	}
	return
}

// DisconnectNetwork .
func (svc *Boar) DisconnectNetwork(ctx context.Context, id, network string) (err error) {
	if err = svc.guest.DisconnectExtraNetwork(ctx, id, network); err != nil {
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
