// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package access

import (
	"context"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

func newIntegrationTest(t *testing.T) (context.Context, *ServerMain, octeliumc.ClientInterface) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	return ctx, NewServerMain(tst.C.OcteliumC), tst.C.OcteliumC
}

func tstCreateSecret(ctx context.Context, t *testing.T, octeliumC octeliumc.ClientInterface) string {
	sec, err := octeliumC.EnterpriseC().CreateSecret(ctx, &enterprisev1.Secret{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &enterprisev1.Secret_Spec{},
		Data: &enterprisev1.Secret_Data{
			Type: &enterprisev1.Secret_Data_Value{
				Value: utilrand.GetRandomString(32),
			},
		},
	})
	assert.Nil(t, err, "%+v", err)

	return sec.Metadata.Name
}

func tstSlackIntegrationSpec(botToken, signingSecret string) *accessv1.Integration_Spec {
	return &accessv1.Integration_Spec{
		Type: &accessv1.Integration_Spec_Slack_{
			Slack: &accessv1.Integration_Spec_Slack{
				BotToken: &accessv1.Integration_Spec_Slack_BotToken{
					Type: &accessv1.Integration_Spec_Slack_BotToken_FromSecret{
						FromSecret: botToken,
					},
				},
				SigningSecret: &accessv1.Integration_Spec_Slack_SigningSecret{
					Type: &accessv1.Integration_Spec_Slack_SigningSecret_FromSecret{
						FromSecret: signingSecret,
					},
				},
			},
		},
	}
}

func tstCreateSlackIntegration(ctx context.Context, t *testing.T,
	srv *ServerMain, octeliumC octeliumc.ClientInterface) *accessv1.Integration {
	item, err := srv.CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: tstSlackIntegrationSpec(
			tstCreateSecret(ctx, t, octeliumC), tstCreateSecret(ctx, t, octeliumC)),
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func tstCreateJiraIntegration(ctx context.Context, t *testing.T,
	octeliumC octeliumc.ClientInterface) *accessv1.Integration {
	item, err := octeliumC.AccessC().CreateIntegration(ctx, &accessv1.Integration{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.Integration_Spec{
			Type: &accessv1.Integration_Spec_Jira_{
				Jira: &accessv1.Integration_Spec_Jira{
					Url:   "https://example.atlassian.net",
					Email: "admin@octelium.com",
					ApiToken: &accessv1.Integration_Spec_Jira_APIToken{
						Type: &accessv1.Integration_Spec_Jira_APIToken_FromSecret{
							FromSecret: tstCreateSecret(ctx, t, octeliumC),
						},
					},
				},
			},
		},
		Status: &accessv1.Integration_Status{
			Id:   utilrand.GetRandomStringCanonical(24),
			Type: accessv1.Integration_Status_JIRA,
		},
	})
	assert.Nil(t, err, "%+v", err)

	return item
}

func TestIntegration(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	item := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	assert.Equal(t, accessv1.Integration_Status_SLACK, item.Status.Type)
	assert.NotEmpty(t, item.Status.Id)
	assert.Equal(t, accessv1.Integration_Status_Synchronization_SYNC_REQUESTED,
		item.Status.Synchronization.State)

	for _, capability := range []accessv1.Integration_Status_Capability{
		accessv1.Integration_Status_NOTIFICATION,
		accessv1.Integration_Status_DIRECT_USER_DELIVERY,
		accessv1.Integration_Status_INTERACTIVE_REVIEW,
		accessv1.Integration_Status_REQUEST_CREATION,
		accessv1.Integration_Status_IDENTITY_RESOLUTION,
		accessv1.Integration_Status_PRESENTATION_UPDATE,
	} {
		assert.Contains(t, item.Status.Capabilities, capability)
	}

	{
		ret, err := srv.GetIntegration(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, item.Metadata.Uid, ret.Metadata.Uid)
	}

	{
		_, err := srv.CreateIntegration(ctx, &accessv1.Integration{
			Metadata: item.Metadata,
			Spec:     item.Spec,
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		itemList, err := srv.ListIntegration(ctx, &accessv1.ListIntegrationOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(itemList.Items))
	}

	{
		_, err := srv.CreateIntegration(ctx, &accessv1.Integration{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: tstSlackIntegrationSpec(utilrand.GetRandomStringCanonical(8),
				tstCreateSecret(ctx, t, octeliumC)),
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateIntegration(ctx, &accessv1.Integration{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.Integration_Spec{},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.UpdateIntegration(ctx, &accessv1.Integration{
			Metadata: &metav1.Metadata{
				Name: item.Metadata.Name,
			},
			Spec: &accessv1.Integration_Spec{
				Type: &accessv1.Integration_Spec_Webhook_{
					Webhook: &accessv1.Integration_Spec_Webhook{
						Url: "https://example.com/hook",
						SigningSecret: &accessv1.Integration_Spec_Webhook_SigningSecret{
							Type: &accessv1.Integration_Spec_Webhook_SigningSecret_FromSecret{
								FromSecret: tstCreateSecret(ctx, t, octeliumC),
							},
						},
					},
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.SynchronizeIntegration(ctx, &accessv1.SynchronizeIntegrationRequest{
			IntegrationRef: umetav1.GetObjectReference(item),
		})
		assert.Nil(t, err, "%+v", err)
	}

	{
		_, err := srv.DeleteIntegration(ctx, &metav1.DeleteOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetIntegration(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.True(t, grpcerr.IsNotFound(err), "%+v", err)
	}
}

func TestIntegrationTarget(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

	item, err := srv.CreateIntegrationTarget(ctx, &accessv1.IntegrationTarget{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.IntegrationTarget_Spec{
			IntegrationRef: umetav1.GetObjectReference(integration),
			Type: &accessv1.IntegrationTarget_Spec_Slack_{
				Slack: &accessv1.IntegrationTarget_Spec_Slack{
					ChannelID: "C12345678",
				},
			},
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessv1.Integration_Status_SLACK, item.Status.Type)

	{
		itemList, err := srv.ListIntegrationTarget(ctx, &accessv1.ListIntegrationTargetOptions{
			IntegrationRef: umetav1.GetObjectReference(integration),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(itemList.Items))
	}

	{
		_, err := srv.CreateIntegrationTarget(ctx, &accessv1.IntegrationTarget{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationTarget_Spec{
				IntegrationRef: umetav1.GetObjectReference(integration),
				Type: &accessv1.IntegrationTarget_Spec_Jira_{
					Jira: &accessv1.IntegrationTarget_Spec_Jira{
						ProjectKey: "OPS",
					},
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateIntegrationTarget(ctx, &accessv1.IntegrationTarget{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationTarget_Spec{
				IntegrationRef: umetav1.GetObjectReference(integration),
				Type: &accessv1.IntegrationTarget_Spec_Slack_{
					Slack: &accessv1.IntegrationTarget_Spec_Slack{},
				},
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		other := tstCreateSlackIntegration(ctx, t, srv, octeliumC)

		next := pbutils.Clone(item).(*accessv1.IntegrationTarget)
		next.Spec.IntegrationRef = umetav1.GetObjectReference(other)

		_, err := srv.UpdateIntegrationTarget(ctx, next)
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		itemG, err := srv.GetIntegrationTarget(ctx, &metav1.GetOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, integration.Metadata.Uid, itemG.Spec.IntegrationRef.Uid)
	}

	{
		_, err := srv.DeleteIntegrationTarget(ctx, &metav1.DeleteOptions{Uid: item.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
	}
}

func TestIntegrationTargetJiraStatuses(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	integration := tstCreateJiraIntegration(ctx, t, octeliumC)

	newTarget := func(approveStatus, rejectStatus string) *accessv1.IntegrationTarget {
		return &accessv1.IntegrationTarget{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationTarget_Spec{
				IntegrationRef: umetav1.GetObjectReference(integration),
				Type: &accessv1.IntegrationTarget_Spec_Jira_{
					Jira: &accessv1.IntegrationTarget_Spec_Jira{
						ProjectKey:    "OPS",
						ApproveStatus: approveStatus,
						RejectStatus:  rejectStatus,
					},
				},
			},
		}
	}

	{
		_, err := srv.CreateIntegrationTarget(ctx, newTarget("Approved", "approved"))
		assert.NotNil(t, err)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		_, err := srv.CreateIntegrationTarget(ctx, newTarget("Approved", "Rejected"))
		assert.Nil(t, err, "%+v", err)
	}
}

func TestIntegrationIdentity(t *testing.T) {
	ctx, srv, octeliumC := newIntegrationTest(t)

	integration := tstCreateSlackIntegration(ctx, t, srv, octeliumC)
	usr := tstCreateUser(ctx, t, octeliumC, "")
	other := tstCreateUser(ctx, t, octeliumC, "")

	externalID := "U12345678"

	item, err := srv.CreateIntegrationIdentity(ctx, &accessv1.IntegrationIdentity{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &accessv1.IntegrationIdentity_Spec{
			IntegrationRef: umetav1.GetObjectReference(integration),
			UserRef:        umetav1.GetObjectReference(usr),
			ExternalID:     externalID,
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Equal(t, accessv1.IntegrationIdentity_Status_MANUAL, item.Status.Source)

	{
		_, err := srv.CreateIntegrationIdentity(ctx, &accessv1.IntegrationIdentity{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationIdentity_Spec{
				IntegrationRef: umetav1.GetObjectReference(integration),
				UserRef:        umetav1.GetObjectReference(other),
				ExternalID:     externalID,
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		_, err := srv.CreateIntegrationIdentity(ctx, &accessv1.IntegrationIdentity{
			Metadata: &metav1.Metadata{
				Name: utilrand.GetRandomStringCanonical(8),
			},
			Spec: &accessv1.IntegrationIdentity_Spec{
				IntegrationRef: umetav1.GetObjectReference(integration),
				UserRef:        umetav1.GetObjectReference(usr),
				ExternalID:     "U87654321",
			},
		})
		assert.NotNil(t, err)
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		itemList, err := srv.ListIntegrationIdentity(ctx, &accessv1.ListIntegrationIdentityOptions{
			UserRef: umetav1.GetObjectReference(usr),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, 1, len(itemList.Items))
	}

	{
		ret, err := srv.ResolveIntegrationIdentity(ctx,
			&accessv1.ResolveIntegrationIdentityRequest{
				IntegrationRef: umetav1.GetObjectReference(integration),
				Type: &accessv1.ResolveIntegrationIdentityRequest_ExternalID{
					ExternalID: externalID,
				},
			})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, ret.IsResolved)
		assert.Equal(t, usr.Metadata.Uid, ret.UserRef.Uid)
		assert.Equal(t, accessv1.IntegrationIdentity_Status_MANUAL, ret.Source)
	}

	{
		ret, err := srv.ResolveIntegrationIdentity(ctx,
			&accessv1.ResolveIntegrationIdentityRequest{
				IntegrationRef: umetav1.GetObjectReference(integration),
				Type: &accessv1.ResolveIntegrationIdentityRequest_UserRef{
					UserRef: umetav1.GetObjectReference(other),
				},
			})
		assert.Nil(t, err, "%+v", err)
		assert.False(t, ret.IsResolved)
		assert.NotEmpty(t, ret.Detail)
	}

	{
		_, err := srv.DeleteIntegrationIdentity(ctx, &metav1.DeleteOptions{
			Uid: item.Metadata.Uid,
		})
		assert.Nil(t, err, "%+v", err)
	}
}

func tstCreateUser(ctx context.Context, t *testing.T,
	octeliumC octeliumc.ClientInterface, email string) *corev1.User {
	usr, err := octeliumC.CoreC().CreateUser(ctx, &corev1.User{
		Metadata: &metav1.Metadata{
			Name: utilrand.GetRandomStringCanonical(8),
		},
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: email,
		},
		Status: &corev1.User_Status{},
	})
	assert.Nil(t, err, "%+v", err)

	return usr
}
