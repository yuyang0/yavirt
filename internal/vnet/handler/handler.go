package handler

import (
	"github.com/projecteru2/yavirt/internal/meta"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	"github.com/projecteru2/yavirt/internal/vnet/types"
)

// Handler .
type Handler interface {
	AssignIP() (meta.IP, error)
	ReleaseIPs(ips ...meta.IP) error
	QueryIPv4(ipv4 string) (meta.IP, error)
	QueryIPs(meta.IPNets) ([]meta.IP, error)

	CreateEndpointNetwork(types.EndpointArgs) (types.EndpointArgs, func(), error)
	JoinEndpointNetwork(types.EndpointArgs) (func(), error)
	DeleteEndpointNetwork(types.EndpointArgs) error

	GetEndpointDevice(devName string) (device.VirtLink, error)

	CreateNetworkPolicy(map[string]string) error
	DeleteNetworkPolicy(map[string]string) error
}
