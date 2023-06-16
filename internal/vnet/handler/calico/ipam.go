package calico

import (
	"context"
	"net"

	"github.com/projecteru2/yavirt/internal/meta"
	calinet "github.com/projecteru2/yavirt/internal/vnet/calico"
	"github.com/projecteru2/yavirt/internal/vnet/ipam"
	"github.com/projecteru2/yavirt/pkg/errors"
)

// NewIP .
func (h *Handler) NewIP(_, cidr string) (meta.IP, error) {
	return calinet.ParseCIDR(cidr)
}

// AssignIP .
func (h *Handler) AssignIP(poolName string) (ip meta.IP, err error) {
	h.Lock()
	defer h.Unlock()
	return h.assignIP(poolName)
}

func (h *Handler) assignIP(poolName string) (ip meta.IP, err error) {
	if ip, err = h.ipam().Assign(context.Background(), poolName); err != nil {
		return nil, errors.Trace(err)
	}

	var roll = ip
	defer func() {
		if err != nil && roll != nil {
			if re := h.releaseIPs(roll); re != nil {
				err = errors.Wrap(err, re)
			}
		}
	}()

	_, gwIPNet, err := net.ParseCIDR("169.254.1.1/32")
	ip.BindGatewayIPNet(gwIPNet)
	return ip, err
}

// ReleaseIPs .
func (h *Handler) ReleaseIPs(ips ...meta.IP) error {
	h.Lock()
	defer h.Unlock()
	return h.releaseIPs(ips...)
}

func (h *Handler) releaseIPs(ips ...meta.IP) error {
	return h.ipam().Release(context.Background(), ips...)
}

// QueryIPs .
func (h *Handler) QueryIPs(ipns meta.IPNets) ([]meta.IP, error) {
	return h.ipam().Query(context.Background(), ipns)
}

func (h *Handler) ipam() ipam.Ipam {
	return h.cali.Ipam()
}

// QueryIPv4 .
func (h *Handler) QueryIPv4(_ string) (meta.IP, error) {
	return nil, errors.Trace(errors.ErrNotImplemented)
}
