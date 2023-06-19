package calico

import (
	"context"
	"fmt"

	apiv3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	libcalierr "github.com/projectcalico/calico/libcalico-go/lib/errors"
	libcaliopt "github.com/projectcalico/calico/libcalico-go/lib/options"

	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/store/etcd"
)

const (
	policyName = "deny-namespaces"
)

// WorkloadEndpoint .
type Policy struct {
	*Driver
}

func newPolicy(driver *Driver) *Policy {
	return &Policy{Driver: driver}
}

// Get .
func (we *Policy) Get(ns string) (cwe *apiv3.NetworkPolicy, err error) {
	we.Lock()
	defer we.Unlock()
	etcd.RetryTimedOut(func() error { //nolint
		cwe, err = we.NetworkPolicies().Get(context.Background(), ns, policyName, libcaliopt.GetOptions{})
		if err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); ok { //nolint
				err = errors.Annotatef(errors.ErrCalicoEndpointNotExists, "%s on %s", ns, policyName)
			}
		}

		return err
	}, 3)
	return
}

// Create .
func (we *Policy) Create(ns string) (cwe *apiv3.NetworkPolicy, err error) {
	we.Lock()
	defer we.Unlock()

	cwe = we.genCalicoNetworkPolicy(ns)

	err = etcd.RetryTimedOut(func() error {
		var created, ce = we.NetworkPolicies().Create(context.Background(), cwe, libcaliopt.SetOptions{})
		if ce != nil {
			if _, ok := ce.(libcalierr.ErrorResourceAlreadyExists); !ok {
				return ce
			}
		}

		cwe = created

		return nil
	}, 3)

	return
}

// Delete .
func (we *Policy) Delete(ns string) error {
	we.Lock()
	defer we.Unlock()

	return etcd.RetryTimedOut(func() error {
		_, err := we.NetworkPolicies().Delete(
			context.Background(),
			ns,
			policyName,
			libcaliopt.DeleteOptions{},
		)
		if err != nil {
			if _, ok := err.(libcalierr.ErrorResourceDoesNotExist); !ok {
				return err
			}
		}
		return nil
	}, 3)
}

// apiVersion: projectcalico.org/v3
// kind: NetworkPolicy
// metadata:
//
//	name: allow-pool2
//	namespace: testpool2
//
// spec:
//
//	types:
//	  - Ingress
//	  - Egress
//	ingress:
//	  - action: Allow
//	    source:
//	      selector: projectcalico.org/namespace == 'testpool2'
//	  - action: Allow
//	    source:
//	      notNets:
//	        - '10.0.0.0/8'
//	egress:
//	  - action: Allow
//	    destination:
//	      selector: projectcalico.org/namespace == 'testpool2'
//	  - action: Allow
//	    destination:
//	      notNets:
//	        - '10.0.0.0/8'
func (we *Policy) genCalicoNetworkPolicy(ns string) *apiv3.NetworkPolicy {
	p := apiv3.NewNetworkPolicy()
	p.Name = policyName

	p.ObjectMeta.Namespace = ns
	p.Spec.Types = []apiv3.PolicyType{apiv3.PolicyTypeIngress, apiv3.PolicyTypeEgress}
	p.Spec.Ingress = []apiv3.Rule{
		{
			Action: apiv3.Allow,
			Source: apiv3.EntityRule{
				Selector: fmt.Sprintf("projectcalico.org/namespace == '%s'", ns),
			},
		},
		{
			Action: apiv3.Allow,
			Source: apiv3.EntityRule{
				NotNets: []string{
					"10.0.0.0/8",
					"192.168.0.0/16",
				},
			},
		},
	}
	p.Spec.Egress = []apiv3.Rule{
		{
			Action: apiv3.Allow,
			Destination: apiv3.EntityRule{
				Selector: fmt.Sprintf("projectcalico.org/namespace == '%s'", ns),
			},
		},
		{
			Action: apiv3.Allow,
			Destination: apiv3.EntityRule{
				NotNets: []string{
					"10.0.0.0/8",
					"192.168.0.0/16",
				},
			},
		},
	}
	return p
}
