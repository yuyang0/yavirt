package boar

import (
	"os"

	"strings"

	pb "github.com/projecteru2/core/rpc/gen"

	"github.com/projecteru2/libyavirt/types"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/models"
	"github.com/projecteru2/yavirt/internal/vnet"
	"github.com/projecteru2/yavirt/internal/vnet/calico"
	"github.com/projecteru2/yavirt/internal/vnet/device"
	calihandler "github.com/projecteru2/yavirt/internal/vnet/handler/calico"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/netx"
)

func (svc *Boar) setupCalico() error {
	if !svc.couldSetupCalico() {
		if svc.Host.NetworkMode == vnet.NetworkCalico {
			return errors.Annotatef(errors.ErrInvalidValue, "invalid Calico config")
		}
		return nil
	}

	if err := svc.setupCalicoHandler(); err != nil {
		return errors.Trace(err)
	}

	// if err := svc.caliHandler.InitGateway(configs.Conf.Calico.GatewayName); err != nil {
	// 	return errors.Trace(err)
	// }

	return nil
}

func (svc *Boar) setupCalicoHandler() error {
	cali, err := calico.NewDriver(configs.Conf.Calico.ConfigFile, configs.Conf.Calico.PoolNames)
	if err != nil {
		return errors.Trace(err)
	}

	dev, err := device.New()
	if err != nil {
		return errors.Trace(err)
	}

	outboundIP, err := netx.GetOutboundIP(configs.Conf.Core.Addrs[0])
	if err != nil {
		return errors.Trace(err)
	}

	svc.caliHandler = calihandler.New(dev, cali, configs.Conf.Calico.PoolNames, outboundIP)

	return nil
}

func (svc *Boar) couldSetupCalico() bool {
	var env = configs.Conf.Calico.ETCDEnv
	return len(configs.Conf.Calico.ConfigFile) > 0 || len(os.Getenv(env)) > 0
}

func convGuestIDsResp(localIDs []string) []string {
	eruIDs := make([]string, len(localIDs))
	for i, id := range localIDs {
		eruIDs[i] = types.EruID(id)
	}
	return eruIDs
}

func convGuestResp(g *models.Guest) (resp *types.Guest) {
	resp = &types.Guest{}
	resp.ID = types.EruID(g.ID)
	resp.Status = g.Status
	resp.CreateTime = g.CreatedTime
	resp.UpdateTime = g.UpdatedTime
	resp.ImageName = g.ImageName
	resp.ImageUser = g.ImageUser
	resp.CPU = g.CPU
	resp.Mem = g.Memory
	resp.Labels = g.JSONLabels

	if len(g.IPs) > 0 {
		var ips = make([]string, len(g.IPs))
		for i, ip := range g.IPs {
			ips[i] = ip.IPAddr()
		}
		resp.Networks = map[string]string{"IP": strings.Join(ips, ", ")}
	}

	return
}

// ConvSetWorkloadsStatusOptions .
func ConvSetWorkloadsStatusOptions(gss []types.EruGuestStatus) *pb.SetWorkloadsStatusOptions {
	css := make([]*pb.WorkloadStatus, len(gss))
	for i, gs := range gss {
		css[i] = convWorkloadStatus(gs)
	}

	return &pb.SetWorkloadsStatusOptions{
		Status: css,
	}
}

func convWorkloadStatus(gs types.EruGuestStatus) *pb.WorkloadStatus {
	return &pb.WorkloadStatus{
		Id:       gs.EruGuestID,
		Running:  gs.Running,
		Healthy:  gs.Healthy,
		Ttl:      int64(gs.TTL.Seconds()),
		Networks: map[string]string{"IP": gs.GetIPAddrs()},
	}
}
