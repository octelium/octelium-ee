package suite

import (
	"context"
	"testing"
	"time"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func testRscStoreResourceReplacement(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	usr := h.CreateWorkloadUser(t, nil)
	sec := h.CreateEnterpriseSecret(t, h.Name()+h.Name())
	cat := h.CreateCatalog(t, nil, []string{"default"})
	h.Eventually(t, "the original resources to be mirrored", eeharness.IngestionBudget,
		func(ctx context.Context) error {
			return mirroredReplacements(ctx, h, usr, sec, cat)
		})

	restore := h.StopEnterpriseWithin(t, "rscstore", eeharness.PropagationBudget)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	_, err := h.CoreC().DeleteUser(ctx, &metav1.DeleteOptions{Uid: usr.Metadata.Uid})
	cancel()
	require.Nil(t, err)
	ctx, cancel = context.WithTimeout(t.Context(), 30*time.Second)
	_, err = h.EnterpriseC().DeleteSecret(ctx, &metav1.DeleteOptions{Uid: sec.Metadata.Uid})
	cancel()
	require.Nil(t, err)
	ctx, cancel = context.WithTimeout(t.Context(), 30*time.Second)
	_, err = h.AccessC().DeleteCatalog(ctx, &metav1.DeleteOptions{Uid: cat.Metadata.Uid})
	cancel()
	require.Nil(t, err)

	replacementUser := h.CreateUser(t, &corev1.User{
		Metadata: &metav1.Metadata{Name: usr.Metadata.Name},
		Spec:     &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD},
	})
	replacementUser.Metadata.DisplayName = h.Name()
	replacementUser = h.UpdateUser(t, replacementUser)
	require.NotEqual(t, usr.Metadata.Uid, replacementUser.Metadata.Uid)

	ctx, cancel = context.WithTimeout(t.Context(), 30*time.Second)
	replacementSecret, err := h.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{Name: sec.Metadata.Name},
		Spec:     &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{Value: h.Name() + h.Name()},
		},
	})
	cancel()
	require.Nil(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := h.EnterpriseC().DeleteSecret(ctx,
			&metav1.DeleteOptions{Uid: replacementSecret.Metadata.Uid})
		if err != nil && !grpcerr.IsNotFound(err) {
			t.Errorf("Could not delete the replacement Secret: %+v", err)
		}
	})
	require.NotEqual(t, sec.Metadata.Uid, replacementSecret.Metadata.Uid)

	ctx, cancel = context.WithTimeout(t.Context(), 30*time.Second)
	replacementCatalog, err := h.AccessC().CreateCatalog(ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: cat.Metadata.Name},
		Spec:     cat.Spec,
	})
	cancel()
	require.Nil(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := h.AccessC().DeleteCatalog(ctx,
			&metav1.DeleteOptions{Uid: replacementCatalog.Metadata.Uid})
		if err != nil && !grpcerr.IsNotFound(err) {
			t.Errorf("Could not delete the replacement Catalog: %+v", err)
		}
	})
	require.NotEqual(t, cat.Metadata.Uid, replacementCatalog.Metadata.Uid)

	restore()

	t.Run("MissedDeletesDoNotLeaveDuplicateNames", func(t *testing.T) {
		h.Eventually(t, "the mirror to remove old UIDs and ingest the replacements",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				return mirroredReplacements(ctx, h, replacementUser, replacementSecret, replacementCatalog)
			})
	})

	t.Run("TheReconciledResourcesSurviveAnotherRestart", func(t *testing.T) {
		h.RestartEnterprise(t, "rscstore")
		h.Eventually(t, "the reconciled replacements to survive the rscstore restart",
			eeharness.IngestionBudget, func(ctx context.Context) error {
				return mirroredReplacements(ctx, h, replacementUser, replacementSecret, replacementCatalog)
			})
		h.Consistently(t, "the replacement UIDs to stay unique", 5*time.Second,
			func(ctx context.Context) error {
				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				return mirroredReplacements(ctx, h, replacementUser, replacementSecret, replacementCatalog)
			})
	})
}

func mirroredReplacements(ctx context.Context, h *eeharness.H,
	usr *corev1.User, sec *enterprisev1.Secret, cat *accessv1.Catalog) error {
	users, err := h.VisibilityCoreC().ListUser(ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{Query: usr.Metadata.Name},
	})
	if err != nil {
		return err
	}
	if len(users.Items) != 1 || users.GetListResponseMeta().GetTotalCount() != 1 ||
		users.Items[0].Metadata.Uid != usr.Metadata.Uid ||
		users.Items[0].Metadata.DisplayName != usr.Metadata.DisplayName {
		return errors.Errorf("the User mirror has not replaced the old resource")
	}

	secrets, err := h.VisibilityEnterpriseC().ListSecret(ctx, &venterprisev1.ListSecretOptions{
		Common: &vmetav1.CommonListOptions{Query: sec.Metadata.Name},
	})
	if err != nil {
		return err
	}
	if len(secrets.Items) != 1 || secrets.GetListResponseMeta().GetTotalCount() != 1 ||
		secrets.Items[0].Metadata.Uid != sec.Metadata.Uid || secrets.Items[0].Data != nil {
		return errors.Errorf("the Secret mirror has not converged to one redacted resource")
	}

	catalogs, err := h.VisibilityAccessC().ListCatalog(ctx, &vaccessv1.ListCatalogOptions{
		Common: &vmetav1.CommonListOptions{Query: cat.Metadata.Name},
	})
	if err != nil {
		return err
	}
	if len(catalogs.Items) != 1 || catalogs.GetListResponseMeta().GetTotalCount() != 1 ||
		catalogs.Items[0].Metadata.Uid != cat.Metadata.Uid {
		return errors.Errorf("the Catalog mirror has not replaced the old resource")
	}
	return nil
}
