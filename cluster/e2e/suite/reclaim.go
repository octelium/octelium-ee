// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/k8sutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"go.uber.org/zap"
)

const pruneBudget = 3 * time.Minute

var reclaimedServices = []string{
	"llama",
	"opensearch",
	"clickhouse",
	"mongo",
	"mysql8",
	"mysql9",
	"mariadb",
	"s3",
	"nats",
	"redis",
	"pg",
	"pg.production",
}

func testReclaimApply(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx, cancel := h.Ctx(t)
	defer cancel()

	var hostnames []string
	var deleted int

	for _, name := range reclaimedServices {
		svc, err := h.CoreC().GetService(ctx, &metav1.GetOptions{Name: name})
		if err != nil {
			if !grpcerr.IsNotFound(err) {
				t.Errorf("Could not read the Service %s: %+v", name, err)
			}
			continue
		}

		if _, err := h.CoreC().DeleteService(ctx,
			&metav1.DeleteOptions{Uid: svc.Metadata.Uid}); err != nil {
			t.Errorf("Could not delete the Service %s: %+v", name, err)
			continue
		}

		deleted++
		hostnames = append(hostnames,
			h.SvcHostname(svc), k8sutils.GetSvcK8sUpstreamHostname(svc, ""))
	}

	zap.L().Info("Reclaiming the Kubernetes workloads of the applied Services",
		zap.Int("services", deleted))

	for _, hostname := range hostnames {
		h.WaitK8sObjectsGone(t, hostname)
	}

	pruneCtx, pruneCancel := context.WithTimeout(ctx, pruneBudget)
	defer pruneCancel()

	if out, err := h.Output(pruneCtx, "sudo -n k3s crictl rmi --prune"); err != nil {
		zap.L().Warn("Could not prune the unused container images",
			zap.Error(err), zap.ByteString("out", out))
	}

	if out, err := h.Output(pruneCtx, "df -h /"); err == nil {
		zap.L().Info("Node disk usage after the reclaim", zap.ByteString("df", out))
	}
}
