// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package support

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"go.uber.org/zap"
)

const (
	maxEnvironmentValueLen = 500
)

var environmentVersions = []struct {
	pkg string
	key string
}{
	{pkg: "octelium", key: "version.core"},
	{pkg: "octeliumee", key: "version.enterprise"},
	{pkg: "cordium", key: "version.cordium"},
}

func (s *Server) getEnvironment(ctx context.Context, lic *enterprisev1.License) map[string]string {
	ret := make(map[string]string)

	if rgnList, err := s.octeliumC.CoreC().ListRegion(ctx, &rmetav1.ListOptions{}); err == nil {
		var names []string
		for _, rgn := range rgnList.Items {
			if rgn.Metadata == nil {
				continue
			}
			names = append(names, rgn.Metadata.Name)

			if rgn.Metadata.Name != "default" || rgn.Status == nil {
				continue
			}

			for _, itm := range environmentVersions {
				if info, ok := rgn.Status.VersionInfoMap[itm.pkg]; ok && info != nil && info.Version != "" {
					ret[itm.key] = truncateEnvironmentValue(info.Version)
				}
			}
		}
		slices.Sort(names)
		ret["regions"] = truncateEnvironmentValue(
			fmt.Sprintf("%d (%s)", len(names), strings.Join(names, ", ")))
	} else {
		zap.L().Debug("Could not list the Regions of the support environment", zap.Error(err))
	}

	if gwList, err := s.octeliumC.CoreC().ListGateway(ctx, &rmetav1.ListOptions{}); err == nil {
		ret["gateways"] = strconv.Itoa(len(gwList.Items))
	} else {
		zap.L().Debug("Could not list the Gateways of the support environment", zap.Error(err))
	}

	if lic != nil {
		if lic.Uid != "" {
			ret["license.uid"] = truncateEnvironmentValue(lic.Uid)
		}
		ret["license.type"] = lic.Type.String()
		if lic.Plan != nil {
			plan := lic.Plan.Type.String()
			if lic.Plan.Tier != "" {
				plan = fmt.Sprintf("%s %s", plan, lic.Plan.Tier)
			}
			ret["license.plan"] = truncateEnvironmentValue(plan)
		}
		if lic.NotAfter != nil && lic.NotAfter.IsValid() {
			ret["license.expiresAt"] = lic.NotAfter.AsTime().UTC().Format(time.RFC3339)
		}
	}

	return ret
}

func truncateEnvironmentValue(arg string) string {
	arg = strings.TrimSpace(arg)
	if len(arg) <= maxEnvironmentValueLen {
		return arg
	}

	cut := maxEnvironmentValueLen
	for cut > 0 && (arg[cut]&0xC0) == 0x80 {
		cut--
	}

	return arg[:cut]
}
