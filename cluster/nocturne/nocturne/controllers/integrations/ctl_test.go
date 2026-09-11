// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package integrations

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

type integrationTest struct {
	ctx       context.Context
	ctrl      *Controller
	octeliumC octeliumc.ClientInterface
	slackFail *atomic.Bool
	slackURL  string
}

func newIntegrationTest(t *testing.T) *integrationTest {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	fail := &atomic.Bool{}

	slackSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if fail.Load() {
			fmt.Fprint(w, `{"ok":false,"error":"invalid_auth"}`)
			return
		}

		fmt.Fprint(w, `{"ok":true,"team_id":"T00000001","team":"Example"}`)
	}))
	t.Cleanup(slackSrv.Close)

	ctrl, err := NewController(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)

	return &integrationTest{
		ctx:       ctx,
		ctrl:      ctrl,
		octeliumC: tst.C.OcteliumC,
		slackFail: fail,
		slackURL:  slackSrv.URL,
	}
}

func (i *integrationTest) createSecret(t *testing.T) string {
	sec, err := i.octeliumC.EnterpriseC().CreateSecret(i.ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(24),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return sec.Metadata.Name
}

func (i *integrationTest) createIntegration(t *testing.T) *accessv1.Integration {
	item, err := i.octeliumC.AccessC().CreateIntegration(i.ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Slack_{
				Slack: &accessv1.Integration_Spec_Slack{
					BotToken: &accessv1.Integration_Spec_Slack_BotToken{
						Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
							FromSecret: i.createSecret(t),
						},
					},
					SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
						Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
							FromSecret: i.createSecret(t),
						},
					},
					BaseURL: i.slackURL,
				},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:   utilrand.GetRandomStringCanonical(24),
			Type: accessv1.Integration_Status_SLACK,
			Synchronization: &accessv1.Integration_Status_Synchronization{
				CreatedAt: pbutils.Now(),
				State:     accessv1.Integration_Status_Synchronization_SYNC_REQUESTED,
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func (i *integrationTest) getIntegration(t *testing.T, uid string) *accessv1.Integration {
	item, err := i.octeliumC.AccessC().GetIntegration(i.ctx, &rmetav1.GetOptions{Uid: uid})
	assert.Nil(t, err, "%+v", err)

	return item
}

func TestIntegrationSynchronizationSuccess(t *testing.T) {
	i := newIntegrationTest(t)

	item := i.createIntegration(t)
	assert.Nil(t, i.ctrl.OnAdd(i.ctx, item))

	item = i.getIntegration(t, item.Metadata.Uid)
	assert.Equal(t, accessv1.Integration_Status_READY, item.Status.State)
	assert.Equal(t, "T00000001", item.Status.ExternalTenantID)
	assert.Equal(t, "Example", item.Status.ExternalTenantName)
	assert.Equal(t, accessv1.Integration_Status_Synchronization_SUCCESS,
		item.Status.Synchronization.State)
	assert.Equal(t, 1, len(item.Status.LastSynchronizations))
	assert.NotNil(t, item.Status.LastSuccessAt)
	assert.Empty(t, item.Status.LastError)

	assert.Nil(t, i.ctrl.OnAdd(i.ctx, item))
	assert.Equal(t, 1, len(i.getIntegration(t, item.Metadata.Uid).Status.LastSynchronizations))
}

func TestIntegrationSynchronizationFailure(t *testing.T) {
	i := newIntegrationTest(t)

	i.slackFail.Store(true)

	item := i.createIntegration(t)
	assert.Nil(t, i.ctrl.OnAdd(i.ctx, item))

	item = i.getIntegration(t, item.Metadata.Uid)
	assert.Equal(t, accessv1.Integration_Status_ERROR, item.Status.State)
	assert.NotEmpty(t, item.Status.LastError)
	assert.NotNil(t, item.Status.LastFailureAt)
	assert.Equal(t, accessv1.Integration_Status_Synchronization_FAILED,
		item.Status.Synchronization.State)
}

func TestIntegrationDisabledIsSkipped(t *testing.T) {
	i := newIntegrationTest(t)

	item := i.createIntegration(t)
	item.Spec.IsDisabled = true

	item, err := i.octeliumC.AccessC().UpdateIntegration(i.ctx, item)
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, i.ctrl.OnAdd(i.ctx, item))

	item = i.getIntegration(t, item.Metadata.Uid)
	assert.Equal(t, accessv1.Integration_Status_STATE_UNSET, item.Status.State)
}

func TestIntegrationOnDeleteRemovesChildren(t *testing.T) {
	i := newIntegrationTest(t)

	item := i.createIntegration(t)

	usr, err := i.octeliumC.CoreC().CreateUser(i.ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type: corev1.User_Spec_HUMAN,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	identity, err := i.octeliumC.AccessC().CreateIntegrationIdentity(i.ctx,
		&accessv1.IntegrationIdentity{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationIdentity_Spec{
				IntegrationRef: umetav1.GetObjectReference(item),
				UserRef:        umetav1.GetObjectReference(usr),
				ExternalID:     "U00000001",
			},
			Status: &accessv1.IntegrationIdentity_Status{},
		})
	assert.Nil(t, err, "%+v", err)

	target, err := i.octeliumC.AccessC().CreateIntegrationTarget(i.ctx,
		&accessv1.IntegrationTarget{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationTarget_Spec{
				IntegrationRef: umetav1.GetObjectReference(item),
				Type: &accessv1.IntegrationTarget_Spec_Slack_{
					Slack: &accessv1.IntegrationTarget_Spec_Slack{
						ChannelID: "C12345678",
					},
				},
			},
			Status: &accessv1.IntegrationTarget_Status{},
		})
	assert.Nil(t, err, "%+v", err)

	assert.Nil(t, i.ctrl.OnDelete(i.ctx, item))

	_, err = i.octeliumC.AccessC().GetIntegrationIdentity(i.ctx, &rmetav1.GetOptions{
		Uid: identity.Metadata.Uid,
	})
	assert.NotNil(t, err)

	_, err = i.octeliumC.AccessC().GetIntegrationTarget(i.ctx, &rmetav1.GetOptions{
		Uid: target.Metadata.Uid,
	})
	assert.NotNil(t, err)
}
