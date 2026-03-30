// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package crowdstrike

import (
	"context"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/hosts"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/nocturne/nocturne/devicemanager/devicemgrcommon"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/utils/ldflags"
)

type Manager struct {
	opts      *devicemgrcommon.ManagerOpts
	c         *client.CrowdStrikeAPISpecification
	octeliumC octeliumc.ClientInterface
}

func New(ctx context.Context, octeliumC octeliumc.ClientInterface, opts *devicemgrcommon.ManagerOpts) (*Manager, error) {

	spec := opts.DeviceManager.Spec.GetCrowdStrike()

	sec, err := octeliumC.EnterpriseC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: spec.ClientSecret.GetFromSecret(),
	})
	if err != nil {
		return nil, err
	}
	falcon.NewClient(&falcon.ApiConfig{
		Context:      ctx,
		ClientId:     spec.ClientID,
		ClientSecret: uenterprisev1.ToSecret(sec).GetValueStr(),
		Cloud: func() falcon.CloudType {
			switch spec.Region {
			case enterprisev1.DeviceManager_Spec_CrowdStrike_EU_1:
				return falcon.CloudEu1
			case enterprisev1.DeviceManager_Spec_CrowdStrike_US_1:
				return falcon.CloudUs1
			case enterprisev1.DeviceManager_Spec_CrowdStrike_US_2:
				return falcon.CloudUs2
			case enterprisev1.DeviceManager_Spec_CrowdStrike_US_GOV:
				return falcon.CloudUsGov1
			default:
				return falcon.CloudAutoDiscover
			}
		}(),
		Debug: ldflags.IsDev(),
	})
	return &Manager{
		opts:      opts,
		octeliumC: octeliumC,
	}, nil
}

func (c *Manager) CollectAll(ctx context.Context) error {
	_, err := c.c.Hosts.GetDeviceDetailsV2(&hosts.GetDeviceDetailsV2Params{})
	if err != nil {
		return err
	}

	return nil
}

func (c *Manager) ManageDevice(ctx context.Context, dev *corev1.Device) error {

	return nil
}

const cmdLinux = `
sudo /opt/CrowdStrike/falconctl -g --aid | awk -F\" '{print $2}')
`

const cmdMac = `
/Applications/Falcon.app/Contents/Resources/falconctl stats | /usr/bin/grep "agentID:" | /usr/bin/awk '{ print $2 }' | /usr/bin/tr -d "-"
`

const psWindows = `
$(Get-ItemProperty HKLM:\SYSTEM\CurrentControlSet\Services\CSAgent\Sim | % AG | %{ "{0:x2}" -f $_ })" -replace " ",""
`
