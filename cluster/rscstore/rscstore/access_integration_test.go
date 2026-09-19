// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vaccessv1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/stretchr/testify/assert"
)

func newTstIntegration(name string, createdAt time.Time,
	status *accessv1.Integration_Status) *accessv1.Integration {
	return &accessv1.Integration{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindIntegration,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       &accessv1.Integration_Spec{},
		Status:     status,
	}
}

func newTstBinding(name string, createdAt time.Time,
	status *accessv1.IntegrationBinding_Status) *accessv1.IntegrationBinding {
	return &accessv1.IntegrationBinding{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindIntegrationBinding,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       &accessv1.IntegrationBinding_Spec{},
		Status:     status,
	}
}

func TestAccessIntegrationSummaries(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()

	insertRscStoreObject(t, env, &accessv1.Secret{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindSecret,
		Metadata:   newRscStoreMetadata("bot-token", now),
		Spec:       &accessv1.Secret_Spec{},
		Status:     &accessv1.Secret_Status{},
		Data: &accessv1.Secret_Data{
			Type: &accessv1.Secret_Data_Value{
				Value: "xoxb-token",
			},
		},
	})

	{
		resp, err := env.srv.getSummaryAccessSecret(env.ctx, &vaccessv1.GetSecretSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
	}

	tenantID := "T00000001"

	insertRscStoreObject(t, env, newTstIntegration("slack-one", now,
		&accessv1.Integration_Status{
			Id:               vutils.UUIDv4(),
			Type:             accessv1.Integration_Status_SLACK,
			State:            accessv1.Integration_Status_READY,
			ExternalTenantID: tenantID,
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_NOTIFICATION,
				accessv1.Integration_Status_DIRECT_USER_DELIVERY,
				accessv1.Integration_Status_INTERACTIVE_REVIEW,
				accessv1.Integration_Status_REQUEST_CREATION,
				accessv1.Integration_Status_IDENTITY_RESOLUTION,
				accessv1.Integration_Status_PRESENTATION_UPDATE,
			},
			Synchronization: &accessv1.Integration_Status_Synchronization{
				State: accessv1.Integration_Status_Synchronization_SUCCESS,
			},
		}))

	insertRscStoreObject(t, env, newTstIntegration("slack-two", now.Add(time.Second),
		&accessv1.Integration_Status{
			Id:               vutils.UUIDv4(),
			Type:             accessv1.Integration_Status_SLACK,
			State:            accessv1.Integration_Status_DEGRADED,
			ExternalTenantID: tenantID,
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_NOTIFICATION,
			},
			Synchronization: &accessv1.Integration_Status_Synchronization{
				State: accessv1.Integration_Status_Synchronization_SYNCING,
			},
		}))

	insertRscStoreObject(t, env, newTstIntegration("jira-one", now.Add(2*time.Second),
		&accessv1.Integration_Status{
			Id:               vutils.UUIDv4(),
			Type:             accessv1.Integration_Status_JIRA,
			State:            accessv1.Integration_Status_ERROR,
			ExternalTenantID: "example.atlassian.net",
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_NOTIFICATION,
				accessv1.Integration_Status_IDENTITY_RESOLUTION,
			},
			Synchronization: &accessv1.Integration_Status_Synchronization{
				State: accessv1.Integration_Status_Synchronization_FAILED,
			},
		}))

	{
		item := newTstIntegration("webhook-one", now.Add(3*time.Second),
			&accessv1.Integration_Status{
				Id:   vutils.UUIDv4(),
				Type: accessv1.Integration_Status_WEBHOOK,
				Capabilities: []accessv1.Integration_Status_Capability{
					accessv1.Integration_Status_NOTIFICATION,
				},
			})
		item.Spec.IsDisabled = true

		insertRscStoreObject(t, env, item)
	}

	{
		resp, err := env.srv.getSummaryAccessIntegration(env.ctx,
			&vaccessv1.GetIntegrationSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 4, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDisabled)

		assert.EqualValues(t, 2, resp.TotalSlack)
		assert.EqualValues(t, 1, resp.TotalJira)
		assert.EqualValues(t, 1, resp.TotalWebhook)

		assert.EqualValues(t, 1, resp.TotalReady)
		assert.EqualValues(t, 1, resp.TotalDegraded)
		assert.EqualValues(t, 1, resp.TotalError)

		assert.EqualValues(t, 1, resp.TotalSynchronizing)
		assert.EqualValues(t, 1, resp.TotalSynchronizationSuccess)
		assert.EqualValues(t, 1, resp.TotalSynchronizationFailed)

		assert.EqualValues(t, 4, resp.TotalNotification)
		assert.EqualValues(t, 1, resp.TotalDirectUserDelivery)
		assert.EqualValues(t, 1, resp.TotalInteractiveReview)
		assert.EqualValues(t, 1, resp.TotalRequestCreation)
		assert.EqualValues(t, 2, resp.TotalIdentityResolution)
		assert.EqualValues(t, 1, resp.TotalPresentationUpdate)

		assert.EqualValues(t, 2, resp.TotalExternalTenant)
	}

	integrationRef := &metav1.ObjectReference{Name: "slack-one", Uid: vutils.UUIDv4()}
	otherIntegrationRef := &metav1.ObjectReference{Name: "jira-one", Uid: vutils.UUIDv4()}
	userRef := &metav1.ObjectReference{Name: "user-one", Uid: vutils.UUIDv4()}
	otherUserRef := &metav1.ObjectReference{Name: "user-two", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.IntegrationIdentity{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindIntegrationIdentity,
		Metadata:   newRscStoreMetadata("identity-one", now),
		Spec:       &accessv1.IntegrationIdentity_Spec{},
		Status: &accessv1.IntegrationIdentity_Status{
			IntegrationRef: integrationRef,
			UserRef:        userRef,
			ExternalID:     "U00000001",
			Source:         accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
		},
	})

	insertRscStoreObject(t, env, &accessv1.IntegrationIdentity{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindIntegrationIdentity,
		Metadata:   newRscStoreMetadata("identity-two", now.Add(time.Second)),
		Spec:       &accessv1.IntegrationIdentity_Spec{},
		Status: &accessv1.IntegrationIdentity_Status{
			IntegrationRef: otherIntegrationRef,
			UserRef:        otherUserRef,
			ExternalID:     "acc-1",
			Source:         accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
		},
	})

	{
		resp, err := env.srv.getSummaryAccessIntegrationIdentity(env.ctx,
			&vaccessv1.GetIntegrationIdentitySummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalEmailDiscovery)
		assert.EqualValues(t, 2, resp.TotalIntegration)
		assert.EqualValues(t, 2, resp.TotalUser)
	}

	requestRef := &metav1.ObjectReference{Name: "req-one", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, newTstBinding("binding-ready", now,
		&accessv1.IntegrationBinding_Status{
			IntegrationRef:  integrationRef,
			RequestRef:      requestRef,
			Audience:        accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			Purpose:         accessv1.IntegrationBinding_Status_REVIEW_SURFACE,
			InteractionMode: accessv1.Policy_Spec_Rule_Surface_INTERACTIVE,
			State:           accessv1.IntegrationBinding_Status_READY,
			DesiredRevision: "rev-1",
			AppliedRevision: "rev-1",
		}))

	insertRscStoreObject(t, env, newTstBinding("binding-pending", now.Add(time.Second),
		&accessv1.IntegrationBinding_Status{
			IntegrationRef:  integrationRef,
			RequestRef:      requestRef,
			UserRef:         userRef,
			Audience:        accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS,
			Purpose:         accessv1.IntegrationBinding_Status_REVIEW_SURFACE,
			InteractionMode: accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY,
			State:           accessv1.IntegrationBinding_Status_PENDING,
			DesiredRevision: "rev-2",
			Attempts:        3,
		}))

	insertRscStoreObject(t, env, newTstBinding("binding-degraded", now.Add(2*time.Second),
		&accessv1.IntegrationBinding_Status{
			IntegrationRef:  otherIntegrationRef,
			RequestRef:      requestRef,
			UserRef:         otherUserRef,
			Audience:        accessv1.Policy_Spec_Rule_Surface_Destination_REQUESTER,
			Purpose:         accessv1.IntegrationBinding_Status_NOTIFICATION,
			State:           accessv1.IntegrationBinding_Status_DEGRADED,
			DesiredRevision: "rev-3",
			AppliedRevision: "rev-0",
			Attempts:        1,
		}))

	insertRscStoreObject(t, env, newTstBinding("binding-closed", now.Add(3*time.Second),
		&accessv1.IntegrationBinding_Status{
			IntegrationRef:  otherIntegrationRef,
			RequestRef:      requestRef,
			UserRef:         userRef,
			Audience:        accessv1.Policy_Spec_Rule_Surface_Destination_SUBJECT,
			Purpose:         accessv1.IntegrationBinding_Status_NOTIFICATION,
			State:           accessv1.IntegrationBinding_Status_CLOSED,
			DesiredRevision: "rev-4",
			AppliedRevision: "rev-4",
		}))

	{
		resp, err := env.srv.getSummaryAccessIntegrationBinding(env.ctx,
			&vaccessv1.GetIntegrationBindingSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 4, resp.TotalNumber)

		assert.EqualValues(t, 1, resp.TotalPending)
		assert.EqualValues(t, 1, resp.TotalReady)
		assert.EqualValues(t, 1, resp.TotalDegraded)
		assert.EqualValues(t, 1, resp.TotalClosed)

		assert.EqualValues(t, 2, resp.TotalReviewSurface)
		assert.EqualValues(t, 2, resp.TotalNotification)

		assert.EqualValues(t, 1, resp.TotalShared)
		assert.EqualValues(t, 1, resp.TotalReviewers)
		assert.EqualValues(t, 1, resp.TotalRequester)
		assert.EqualValues(t, 1, resp.TotalSubject)

		assert.EqualValues(t, 1, resp.TotalInteractive)

		assert.EqualValues(t, 2, resp.TotalOutOfDate)
		assert.EqualValues(t, 2, resp.TotalFailing)

		assert.EqualValues(t, 2, resp.TotalIntegration)
		assert.EqualValues(t, 1, resp.TotalRequest)
	}
}

func TestAccessIntegrationListFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvAccess{s: env.srv}
	now := time.Now().UTC()

	tenantID := "T00000001"

	insertRscStoreObject(t, env, newTstIntegration("slack-one", now,
		&accessv1.Integration_Status{
			Id:               vutils.UUIDv4(),
			Type:             accessv1.Integration_Status_SLACK,
			State:            accessv1.Integration_Status_READY,
			ExternalTenantID: tenantID,
			Capabilities: []accessv1.Integration_Status_Capability{
				accessv1.Integration_Status_NOTIFICATION,
				accessv1.Integration_Status_INTERACTIVE_REVIEW,
			},
			Synchronization: &accessv1.Integration_Status_Synchronization{
				State: accessv1.Integration_Status_Synchronization_SUCCESS,
			},
		}))

	{
		item := newTstIntegration("jira-one", now.Add(time.Second),
			&accessv1.Integration_Status{
				Id:               vutils.UUIDv4(),
				Type:             accessv1.Integration_Status_JIRA,
				State:            accessv1.Integration_Status_ERROR,
				ExternalTenantID: "example.atlassian.net",
				Capabilities: []accessv1.Integration_Status_Capability{
					accessv1.Integration_Status_NOTIFICATION,
				},
				Synchronization: &accessv1.Integration_Status_Synchronization{
					State: accessv1.Integration_Status_Synchronization_FAILED,
				},
			})
		item.Spec.IsDisabled = true

		insertRscStoreObject(t, env, item)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			Type: accessv1.Integration_Status_SLACK,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "slack-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			State: accessv1.Integration_Status_ERROR,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "jira-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			SynchronizationState: accessv1.Integration_Status_Synchronization_SUCCESS,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "slack-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			Capability: accessv1.Integration_Status_INTERACTIVE_REVIEW,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "slack-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			Capability: accessv1.Integration_Status_NOTIFICATION,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 2)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			IsDisabled: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "jira-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{
			ExternalTenantID: tenantID,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "slack-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegration(env.ctx, &vaccessv1.ListIntegrationOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 2)
	}

	integrationRef := &metav1.ObjectReference{Name: "slack-one", Uid: vutils.UUIDv4()}
	otherIntegrationRef := &metav1.ObjectReference{Name: "jira-one", Uid: vutils.UUIDv4()}
	userRef := &metav1.ObjectReference{Name: "user-one", Uid: vutils.UUIDv4()}
	requestRef := &metav1.ObjectReference{Name: "req-one", Uid: vutils.UUIDv4()}
	otherRequestRef := &metav1.ObjectReference{Name: "req-two", Uid: vutils.UUIDv4()}

	insertRscStoreObject(t, env, &accessv1.IntegrationIdentity{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindIntegrationIdentity,
		Metadata:   newRscStoreMetadata("identity-one", now),
		Spec:       &accessv1.IntegrationIdentity_Spec{},
		Status: &accessv1.IntegrationIdentity_Status{
			IntegrationRef: integrationRef,
			UserRef:        userRef,
			ExternalID:     "U00000001",
			Source:         accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
		},
	})

	insertRscStoreObject(t, env, &accessv1.IntegrationIdentity{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindIntegrationIdentity,
		Metadata:   newRscStoreMetadata("identity-two", now.Add(time.Second)),
		Spec:       &accessv1.IntegrationIdentity_Spec{},
		Status: &accessv1.IntegrationIdentity_Status{
			IntegrationRef: otherIntegrationRef,
			UserRef:        &metav1.ObjectReference{Name: "user-two", Uid: vutils.UUIDv4()},
			ExternalID:     "acc-1",
			Source:         accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
		},
	})

	{
		resp, err := srv.ListIntegrationIdentity(env.ctx,
			&vaccessv1.ListIntegrationIdentityOptions{
				IntegrationRef: integrationRef,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "identity-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationIdentity(env.ctx,
			&vaccessv1.ListIntegrationIdentityOptions{
				UserRef: userRef,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "identity-one", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationIdentity(env.ctx,
			&vaccessv1.ListIntegrationIdentityOptions{
				Source: accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 2)
	}

	insertRscStoreObject(t, env, newTstBinding("binding-ready", now,
		&accessv1.IntegrationBinding_Status{
			IntegrationRef:  integrationRef,
			RequestRef:      requestRef,
			Audience:        accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			Purpose:         accessv1.IntegrationBinding_Status_REVIEW_SURFACE,
			InteractionMode: accessv1.Policy_Spec_Rule_Surface_INTERACTIVE,
			State:           accessv1.IntegrationBinding_Status_READY,
			DesiredRevision: "rev-1",
			AppliedRevision: "rev-1",
		}))

	insertRscStoreObject(t, env, newTstBinding("binding-pending", now.Add(time.Second),
		&accessv1.IntegrationBinding_Status{
			IntegrationRef:  otherIntegrationRef,
			RequestRef:      otherRequestRef,
			UserRef:         userRef,
			Audience:        accessv1.Policy_Spec_Rule_Surface_Destination_REVIEWERS,
			Purpose:         accessv1.IntegrationBinding_Status_NOTIFICATION,
			InteractionMode: accessv1.Policy_Spec_Rule_Surface_DEEP_LINK_ONLY,
			State:           accessv1.IntegrationBinding_Status_PENDING,
			DesiredRevision: "rev-2",
			Attempts:        2,
		}))

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				IntegrationRef: integrationRef,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-ready", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				RequestRef: otherRequestRef,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-pending", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				UserRef: userRef,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-pending", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				State: accessv1.IntegrationBinding_Status_READY,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-ready", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				Purpose: accessv1.IntegrationBinding_Status_REVIEW_SURFACE,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-ready", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				Audience: accessv1.Policy_Spec_Rule_Surface_Destination_SHARED,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-ready", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				InteractionMode: accessv1.Policy_Spec_Rule_Surface_INTERACTIVE,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-ready", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				IsOutOfDate: true,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-pending", resp.Items[0].Metadata.Name)
	}

	{
		resp, err := srv.ListIntegrationBinding(env.ctx,
			&vaccessv1.ListIntegrationBindingOptions{
				IsFailing: true,
			})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "binding-pending", resp.Items[0].Metadata.Name)
	}
}

func TestAccessSecretDataIsNeverStored(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvAccess{s: env.srv}
	now := time.Now().UTC()

	insertRscStoreObject(t, env, &accessv1.Secret{
		ApiVersion: uaccessv1.APIVersion,
		Kind:       uaccessv1.KindSecret,
		Metadata:   newRscStoreMetadata("bot-token", now),
		Spec:       &accessv1.Secret_Spec{},
		Status:     &accessv1.Secret_Status{},
		Data: &accessv1.Secret_Data{
			Type: &accessv1.Secret_Data_Value{
				Value: "xoxb-token",
			},
		},
	})

	resp, err := srv.ListSecret(env.ctx, &vaccessv1.ListSecretOptions{})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "bot-token", resp.Items[0].Metadata.Name)
	assert.Nil(t, resp.Items[0].Data)

	var stored string
	err = env.srv.db.QueryRowContext(env.ctx,
		`SELECT rsc_str FROM resources WHERE kind = ?`, uaccessv1.KindSecret).Scan(&stored)
	assert.Nil(t, err, "%+v", err)
	assert.NotContains(t, stored, "xoxb-token")
}
