// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package harness

import (
	"context"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (h *H) CreateEnterpriseSecret(t *testing.T, value string) *enterprisev1.Secret {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec:     &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{Value: value},
		},
	})
	if err != nil {
		t.Fatalf("Could not create the enterprise Secret: %+v", err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "Secret", ret.Metadata.Name, func(ctx context.Context) error {
			_, err := h.EnterpriseC().DeleteSecret(ctx,
				&metav1.DeleteOptions{Uid: ret.Metadata.Uid})
			return err
		})
	})

	return ret
}

func (h *H) CreateCoreSecret(t *testing.T, value string) *corev1.Secret {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.CoreC().CreateSecret(ctx, &corev1.Secret{
		Metadata: &metav1.Metadata{Name: h.Name()},
		Spec:     &corev1.Secret_Spec{},
		Data: &corev1.Secret_Data{
			Type: &corev1.Secret_Data_Value{Value: value},
		},
	})
	if err != nil {
		t.Fatalf("Could not create the core Secret: %+v", err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "Secret", ret.Metadata.Name, func(ctx context.Context) error {
			_, err := h.CoreC().DeleteSecret(ctx, &metav1.DeleteOptions{Uid: ret.Metadata.Uid})
			return err
		})
	})

	return ret
}

func (h *H) CreateCollectorExporter(t *testing.T,
	exp *enterprisev1.CollectorExporter) *enterprisev1.CollectorExporter {
	t.Helper()

	if exp.Metadata == nil {
		exp.Metadata = &metav1.Metadata{}
	}
	if exp.Metadata.Name == "" {
		exp.Metadata.Name = h.Name()
	}

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().CreateCollectorExporter(ctx, exp)
	if err != nil {
		t.Fatalf("Could not create the CollectorExporter %s: %+v", exp.Metadata.Name, err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "CollectorExporter", ret.Metadata.Name,
			func(ctx context.Context) error {
				_, err := h.EnterpriseC().DeleteCollectorExporter(ctx,
					&metav1.DeleteOptions{Uid: ret.Metadata.Uid})
				return err
			})
	})

	return ret
}

func (h *H) UpdateCollectorExporter(t *testing.T,
	exp *enterprisev1.CollectorExporter) *enterprisev1.CollectorExporter {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().UpdateCollectorExporter(ctx, exp)
	if err != nil {
		t.Fatalf("Could not update the CollectorExporter %s: %+v", exp.Metadata.Name, err)
	}

	return ret
}

func (h *H) CreateDirectoryProvider(t *testing.T,
	dp *enterprisev1.DirectoryProvider) *enterprisev1.DirectoryProvider {
	t.Helper()

	if dp.Metadata == nil {
		dp.Metadata = &metav1.Metadata{}
	}
	if dp.Metadata.Name == "" {
		dp.Metadata.Name = h.Name()
	}

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().CreateDirectoryProvider(ctx, dp)
	if err != nil {
		t.Fatalf("Could not create the DirectoryProvider %s: %+v", dp.Metadata.Name, err)
	}

	t.Cleanup(func() {
		h.deleteQuietly(t, "DirectoryProvider", ret.Metadata.Name,
			func(ctx context.Context) error {
				_, err := h.EnterpriseC().DeleteDirectoryProvider(ctx,
					&metav1.DeleteOptions{Uid: ret.Metadata.Uid})
				return err
			})
	})

	return ret
}

func (h *H) UpdateGroup(t *testing.T, grp *corev1.Group) *corev1.Group {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.CoreC().UpdateGroup(ctx, grp)
	if err != nil {
		t.Fatalf("Could not update the Group %s: %+v", grp.Metadata.Name, err)
	}

	return ret
}

func (h *H) EnterpriseClusterConfig(t *testing.T) *enterprisev1.ClusterConfig {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().GetClusterConfig(ctx,
		&enterprisev1.GetClusterConfigRequest{})
	if err != nil {
		t.Fatalf("Could not get the enterprise ClusterConfig: %+v", err)
	}

	return ret
}

func (h *H) UpdateEnterpriseClusterConfig(t *testing.T,
	cc *enterprisev1.ClusterConfig) *enterprisev1.ClusterConfig {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().UpdateClusterConfig(ctx, cc)
	if err != nil {
		t.Fatalf("Could not update the enterprise ClusterConfig: %+v", err)
	}

	return ret
}

func (h *H) SetCollectorPipelines(t *testing.T,
	pipelines ...*enterprisev1.ClusterConfig_Spec_Collector_Pipeline) {
	t.Helper()

	cc := h.EnterpriseClusterConfig(t)

	before := cc.Spec.Collector
	t.Cleanup(func() {
		cur := h.EnterpriseClusterConfig(t)
		cur.Spec.Collector = before
		if _, err := h.EnterpriseC().UpdateClusterConfig(context.Background(), cur); err != nil {
			zap.L().Warn("Could not restore the ClusterConfig Collector pipelines",
				zap.Error(err))
		}
	})

	if cc.Spec.Collector == nil {
		cc.Spec.Collector = &enterprisev1.ClusterConfig_Spec_Collector{}
	}
	cc.Spec.Collector.Pipelines = append(cc.Spec.Collector.Pipelines, pipelines...)

	h.UpdateEnterpriseClusterConfig(t, cc)
}

func (h *H) SecretStore(t *testing.T, name string) *enterprisev1.SecretStore {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().GetSecretStore(ctx, &metav1.GetOptions{Name: name})
	if err != nil {
		t.Fatalf("Could not get the SecretStore %s: %+v", name, err)
	}

	return ret
}

func (h *H) SetSecretStoreSpec(t *testing.T,
	name string, spec *enterprisev1.SecretStore_Spec) *enterprisev1.SecretStore {
	t.Helper()

	ss := h.SecretStore(t, name)
	ss.Spec = spec

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().UpdateSecretStore(ctx, ss)
	if err != nil {
		t.Fatalf("Could not update the SecretStore %s: %+v", name, err)
	}

	return ret
}

func (h *H) SynchronizeSecretStore(t *testing.T, ss *enterprisev1.SecretStore) {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	_, err := h.EnterpriseC().SynchronizeSecretStore(ctx,
		&enterprisev1.SynchronizeSecretStoreRequest{
			SecretStoreRef: umetav1.GetObjectReference(ss),
		})
	if err != nil {
		t.Fatalf("Could not synchronize the SecretStore %s: %+v", ss.Metadata.Name, err)
	}
}

func (h *H) WaitSecretStoreSynchronized(t *testing.T,
	name string, typ enterprisev1.SecretStore_Status_Type, budget time.Duration) {
	t.Helper()

	h.Eventually(t, "the SecretStore synchronization to complete", budget,
		func(ctx context.Context) error {
			ss, err := h.EnterpriseC().GetSecretStore(ctx, &metav1.GetOptions{Name: name})
			if err != nil {
				return err
			}

			if ss.Status.Type != typ {
				return errors.Errorf("the SecretStore type is %s, want %s", ss.Status.Type, typ)
			}
			if ss.Status.Synchronization != nil {
				return errors.Errorf("the SecretStore is still %s",
					ss.Status.Synchronization.State)
			}
			if len(ss.Status.LastSynchronizations) < 1 {
				return errors.Errorf("the SecretStore has no completed synchronization yet")
			}

			last := ss.Status.LastSynchronizations[0]
			if last.State != enterprisev1.SecretStore_Status_Synchronization_SUCCESS {
				return errors.Errorf("the last synchronization is %s", last.State)
			}
			if ss.Status.State != enterprisev1.SecretStore_Status_OK {
				return errors.Errorf("the SecretStore state is %s", ss.Status.State)
			}

			return nil
		})
}

func (h *H) SynchronizeDirectoryProvider(t *testing.T, dp *enterprisev1.DirectoryProvider) {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	_, err := h.EnterpriseC().SynchronizeDirectoryProvider(ctx,
		&enterprisev1.SynchronizeDirectoryProviderRequest{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
	if err != nil {
		t.Fatalf("Could not synchronize the DirectoryProvider %s: %+v",
			dp.Metadata.Name, err)
	}
}

func (h *H) WaitDirectorySynchronized(t *testing.T,
	dp *enterprisev1.DirectoryProvider,
	want enterprisev1.DirectoryProvider_Status_Synchronization_State,
	budget time.Duration) {
	t.Helper()

	before := len(dp.Status.LastSynchronizations)

	h.Eventually(t, "the DirectoryProvider synchronization to complete", budget,
		func(ctx context.Context) error {
			cur, err := h.EnterpriseC().GetDirectoryProvider(ctx,
				&metav1.GetOptions{Uid: dp.Metadata.Uid})
			if err != nil {
				return err
			}

			if cur.Status.Synchronization != nil {
				return errors.Errorf("the DirectoryProvider is still %s",
					cur.Status.Synchronization.State)
			}
			if len(cur.Status.LastSynchronizations) <= before {
				return errors.Errorf("no new synchronization has completed yet")
			}
			if got := cur.Status.LastSynchronizations[0].State; got != want {
				return errors.Errorf("the last synchronization is %s, want %s", got, want)
			}

			return nil
		})
}

func (h *H) DirectoryUsers(t *testing.T,
	dp *enterprisev1.DirectoryProvider) []*enterprisev1.DirectoryProviderUser {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().ListDirectoryProviderUser(ctx,
		&enterprisev1.ListDirectoryProviderUserOptions{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
	if err != nil {
		t.Fatalf("Could not list the DirectoryProvider Users: %+v", err)
	}

	return ret.Items
}

func (h *H) DirectoryGroups(t *testing.T,
	dp *enterprisev1.DirectoryProvider) []*enterprisev1.DirectoryProviderGroup {
	t.Helper()

	ctx, cancel := h.Ctx(t)
	defer cancel()

	ret, err := h.EnterpriseC().ListDirectoryProviderGroup(ctx,
		&enterprisev1.ListDirectoryProviderGroupOptions{
			DirectoryProviderRef: umetav1.GetObjectReference(dp),
		})
	if err != nil {
		t.Fatalf("Could not list the DirectoryProvider Groups: %+v", err)
	}

	return ret.Items
}

func (h *H) deleteQuietly(t *testing.T, kind, name string, fn func(ctx context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := fn(ctx); err != nil {
		if grpcerr.IsNotFound(err) {
			return
		}
		zap.L().Warn("Could not clean up fixture",
			zap.String("kind", kind), zap.String("name", name), zap.Error(err))
	}
}
