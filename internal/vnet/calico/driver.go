package calico

import (
	"context"
	"strings"
	"sync"

	calitype "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	"github.com/projectcalico/calico/libcalico-go/lib/options"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Driver .
type Driver struct {
	sync.Mutex

	clientv3.Interface

	poolNames map[string]struct{}
}

// NewDriver .
func NewDriver(configFile string, poolNames []string) (*Driver, error) {
	caliConf, err := apiconfig.LoadClientConfig(configFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	cali, err := clientv3.New(*caliConf)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var driver = &Driver{
		Interface: cali,
		poolNames: map[string]struct{}{},
	}

	for _, pn := range poolNames {
		driver.poolNames[pn] = struct{}{}
	}

	return driver, nil
}

func (d *Driver) getIPPool(poolName string) (pool *calitype.IPPool, err error) {
	if poolName != "" {
		return d.IPPools().Get(context.Background(), poolName, options.GetOptions{})
	}
	pools, err := d.IPPools().List(context.Background(), options.ListOptions{})
	switch {
	case err != nil:
		return pool, errors.Trace(err)
	case len(pools.Items) < 1:
		return pool, errors.Trace(errors.ErrCalicoPoolNotExists)
	}

	if len(d.poolNames) < 1 {
		return &pools.Items[0], nil
	}

	for _, p := range pools.Items {
		if _, exists := d.poolNames[p.Name]; exists {
			return &p, nil
		}
	}

	return pool, errors.Annotatef(errors.ErrCalicoPoolNotExists, "no such pool names: %s", d.poolNamesStr())
}

func (d *Driver) poolNamesStr() string {
	var s = make([]string, len(d.poolNames), 0)
	for name := range d.poolNames {
		s = append(s, name)
	}
	return strings.Join(s, ", ")
}

// Ipam .
func (d *Driver) Ipam() *Ipam {
	return newIpam(d)
}

// WorkloadEndpoint .
func (d *Driver) WorkloadEndpoint() *WorkloadEndpoint {
	return newWorkloadEndpoint(d)
}

func (d *Driver) Policy() *Policy {
	return newPolicy(d)
}
