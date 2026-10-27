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
	"strings"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	eescenario "github.com/octelium/octelium-ee/cluster/e2e/scenario"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/cluster/e2e/scenario"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const upgradeBudget = 25 * time.Minute

func testLicenseLifecycle(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	t.Cleanup(func() {
		h.ClusterC().DeleteLicense(context.Background(), &enterprisev1.DeleteLicenseRequest{})
	})

	t.Run("NoneOnAFreshCluster", func(t *testing.T) {
		res, err := h.ClusterC().GetLicense(ctx, &enterprisev1.GetLicenseRequest{})
		require.Nil(t, err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_NONE, res.State)
	})

	t.Run("MalformedIsRefused", func(t *testing.T) {
		_, err := h.ClusterC().SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: "not-a-jwt",
		})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	t.Run("EmptyIsRefused", func(t *testing.T) {
		_, err := h.ClusterC().SetLicense(ctx, &enterprisev1.SetLicenseRequest{})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err))
	})

	t.Run("UnsignedIsRefused", func(t *testing.T) {
		_, err := h.ClusterC().SetLicense(ctx, &enterprisev1.SetLicenseRequest{
			Jwt: strings.Repeat("a.", 2) + "b",
		})
		assert.NotNil(t, err)
	})

	t.Run("DeleteIsIdempotent", func(t *testing.T) {
		_, err := h.ClusterC().DeleteLicense(ctx, &enterprisev1.DeleteLicenseRequest{})
		assert.Nil(t, err)

		res, err := h.ClusterC().GetLicense(ctx, &enterprisev1.GetLicenseRequest{})
		require.Nil(t, err)
		assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_NONE, res.State)
	})

	t.Run("TheClusterConfigAgrees", func(t *testing.T) {
		cc := h.EnterpriseClusterConfig(t)
		if cc.Status.LicenseInfo != nil {
			assert.Equal(t, enterprisev1.ClusterConfig_Status_LicenseInfo_NONE,
				cc.Status.LicenseInfo.State)
		}
	})
}

func testClusterInfo(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	res, err := h.ClusterC().GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
	require.Nil(t, err)
	require.NotNil(t, res.PackageEnterprise)

	t.Run("TheRegionCarriesTheEnterpriseVersion", func(t *testing.T) {
		rgn, err := h.CoreC().GetRegion(ctx, &metav1.GetOptions{Name: "default"})
		require.Nil(t, err)
		require.NotNil(t, rgn.Status.VersionInfoMap)

		info, ok := rgn.Status.VersionInfoMap[eescenario.Package]
		require.True(t, ok)
		assert.Equal(t, eescenario.Package, info.Package)
		assert.NotNil(t, info.SetAt)
	})
}

func testClusterUpgrade(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, scenario.CapUpgrade)

	ctx := t.Context()

	res, err := h.ClusterC().GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
	require.Nil(t, err)
	require.NotNil(t, res.PackageEnterprise)

	before := h.EnterpriseClusterConfig(t)
	beforeCount := before.Status.TotalSuccessfulUpgrades

	_, err = h.ClusterC().UpgradeCluster(ctx, &enterprisev1.UpgradeClusterRequest{
		Request: &enterprisev1.UpgradeClusterRequest_Request{
			PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{
				Version: res.PackageEnterprise.CurrentVersion,
			},
		},
	})
	require.Nil(t, err)

	h.Eventually(t, "the upgrade to complete", upgradeBudget,
		func(ctx context.Context) error {
			cc, err := h.EnterpriseC().GetClusterConfig(ctx,
				&enterprisev1.GetClusterConfigRequest{})
			if err != nil {
				return err
			}
			if cc.Status.TotalSuccessfulUpgrades > beforeCount {
				return nil
			}
			if cc.Status.UpgradeRequest == nil {
				return errors.Errorf("no upgrade request is in flight")
			}
			return errors.Errorf("the upgrade is %s", cc.Status.UpgradeRequest.State)
		})

	t.Run("EveryComponentIsBackUp", func(t *testing.T) {
		for _, name := range eescenario.Deployments {
			h.MustWaitDeployment(t, name)
		}
	})

	t.Run("NothingWasLost", func(t *testing.T) {
		_, err := h.EnterpriseC().ListSecret(ctx, &enterprisev1.ListSecretOptions{})
		assert.Nil(t, err)

		waitAccessLogGrows(t, newTrafficDriver(t, h), 1)
	})
}

func testClusterManagerUpgradeLifecycle(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)
	h.Require(t, scenario.CapUpgrade)
	ctx := t.Context()
	info, err := h.ClusterC().GetClusterInfo(ctx, &enterprisev1.GetClusterInfoRequest{})
	require.NoError(t, err)
	require.NotNil(t, info.PackageEnterprise)
	before := h.EnterpriseClusterConfig(t)
	require.Nil(t, before.Status.UpgradeRequest)
	jobsBefore, err := h.K8sC().BatchV1().Jobs(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{LabelSelector: "octelium.com/component=genesis"})
	require.NoError(t, err)
	existing := map[types.UID]bool{}
	for _, job := range jobsBefore.Items {
		existing[job.UID] = true
	}
	request := &enterprisev1.UpgradeClusterRequest{Request: &enterprisev1.UpgradeClusterRequest_Request{
		PackageEnterprise: &enterprisev1.UpgradeClusterRequest_Request_PackageEnterprise{Version: info.PackageEnterprise.CurrentVersion},
	}}
	_, err = h.ClusterC().UpgradeCluster(ctx, request)
	require.NoError(t, err)
	_, err = h.ClusterC().UpgradeCluster(ctx, request)
	require.Error(t, err)
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	h.Eventually(t, "ClusterManager to persist the successful upgrade", upgradeBudget, func(ctx context.Context) error {
		cc, err := h.EnterpriseC().GetClusterConfig(ctx, &enterprisev1.GetClusterConfigRequest{})
		if err != nil {
			return err
		}
		if cc.Status.TotalFailedUpgrades != before.Status.TotalFailedUpgrades {
			return errors.Errorf("the upgrade failed")
		}
		if cc.Status.UpgradeRequest != nil || cc.Status.TotalSuccessfulUpgrades != before.Status.TotalSuccessfulUpgrades+1 {
			return errors.Errorf("the upgrade has not finished")
		}
		return nil
	})
	after := h.EnterpriseClusterConfig(t)
	require.NotEmpty(t, after.Status.LastUpgradeRequests)
	history := after.Status.LastUpgradeRequests[0]
	assert.Equal(t, enterprisev1.ClusterConfig_Status_UpgradeRequest_SUCCESS, history.State)
	require.NotNil(t, history.Request)
	require.NotNil(t, history.Request.PackageEnterprise)
	assert.Equal(t, info.PackageEnterprise.CurrentVersion, history.Request.PackageEnterprise.Version)
	assert.Nil(t, history.Request.Core)
	assert.Nil(t, history.Request.PackageCordium)
	require.NotNil(t, history.CreatedAt)
	require.NotNil(t, history.DoneAt)
	assert.False(t, history.DoneAt.AsTime().Before(history.CreatedAt.AsTime()))
	assert.LessOrEqual(t, len(after.Status.LastUpgradeRequests), 200)
	jobsAfter, err := h.K8sC().BatchV1().Jobs(vutils.K8sNS).List(ctx, k8smetav1.ListOptions{LabelSelector: "octelium.com/component=genesis"})
	require.NoError(t, err)
	completed := 0
	for _, job := range jobsAfter.Items {
		if existing[job.UID] {
			continue
		}
		require.Len(t, job.Spec.Template.Spec.Containers, 1)
		assert.Equal(t, "octeliumee-genesis", job.Spec.Template.Spec.Containers[0].Name)
		complete := job.Status.Succeeded > 0
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete && condition.Status == k8scorev1.ConditionTrue {
				complete = true
			}
		}
		if complete {
			completed++
		}
	}
	assert.Equal(t, 1, completed)
	h.MustWaitDeployment(t, "octeliumee-clusterman")
}
