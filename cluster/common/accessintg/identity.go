// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"fmt"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
)

type Resolution struct {
	IsResolved bool

	User       *corev1.User
	ExternalID string

	Identity *accessv1.IntegrationIdentity
	Source   accessv1.IntegrationIdentity_Status_Source

	Detail string
}

type ResolveOpts struct {
	OcteliumC   octeliumc.ClientInterface
	Integration *accessv1.Integration
	Resolver    IdentityResolver
}

func ResolveUserFromExternalID(ctx context.Context, opts *ResolveOpts,
	externalID string) (*Resolution, error) {
	externalID = strings.TrimSpace(externalID)
	if externalID == "" {
		return &Resolution{
			Detail: "No external actor identifier was provided",
		}, nil
	}

	identity, err := getIdentityByExternalID(ctx, opts.OcteliumC, opts.Integration, externalID)
	if err != nil {
		return nil, err
	}

	if identity != nil {
		usr, err := accesscmd.GetUser(ctx, opts.OcteliumC, identity.Status.UserRef)
		if err != nil {
			return nil, err
		}
		if usr == nil {
			return &Resolution{
				Detail: fmt.Sprintf(
					"The IntegrationIdentity %s refers to a User that no longer exists",
					identity.Metadata.Name),
			}, nil
		}

		return &Resolution{
			IsResolved: true,
			User:       usr,
			ExternalID: externalID,
			Identity:   identity,
			Source:     identity.Status.Source,
		}, nil
	}

	if !isEmailDiscoveryEnabled(opts.Integration) {
		return &Resolution{
			Detail: "There is no IntegrationIdentity for this external actor and the email discovery is disabled",
		}, nil
	}

	if opts.Resolver == nil {
		return &Resolution{
			Detail: "This Integration cannot resolve its own external actors",
		}, nil
	}

	externalUsr, err := opts.Resolver.GetExternalUser(ctx, externalID)
	if err != nil {
		return nil, err
	}

	if externalUsr == nil || externalUsr.Email == "" {
		return &Resolution{
			Detail: "The provider does not report a verified email for this external actor",
		}, nil
	}

	usr, err := getUserByEmail(ctx, opts.OcteliumC, externalUsr.Email)
	if err != nil {
		return nil, err
	}
	if usr == nil {
		return &Resolution{
			Detail: fmt.Sprintf("No unique User has the email %s", externalUsr.Email),
		}, nil
	}

	identity, err = setDiscoveredIdentity(ctx, opts.OcteliumC, opts.Integration, usr, externalUsr)
	if err != nil {
		return nil, err
	}

	return &Resolution{
		IsResolved: true,
		User:       usr,
		ExternalID: externalID,
		Identity:   identity,
		Source:     accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
	}, nil
}

func ResolveExternalIDFromUser(ctx context.Context, opts *ResolveOpts,
	usr *corev1.User) (*Resolution, error) {
	if usr == nil {
		return &Resolution{
			Detail: "No User was provided",
		}, nil
	}

	identity, err := getIdentityByUserUID(ctx, opts.OcteliumC, opts.Integration, usr.Metadata.Uid)
	if err != nil {
		return nil, err
	}

	if identity != nil {
		return &Resolution{
			IsResolved: true,
			User:       usr,
			ExternalID: identity.Status.ExternalID,
			Identity:   identity,
			Source:     identity.Status.Source,
		}, nil
	}

	if !isEmailDiscoveryEnabled(opts.Integration) {
		return &Resolution{
			Detail: "There is no IntegrationIdentity for this User and the email discovery is disabled",
		}, nil
	}

	if opts.Resolver == nil {
		return &Resolution{
			Detail: "This Integration cannot resolve its own external actors",
		}, nil
	}

	if usr.Spec == nil || usr.Spec.Email == "" {
		return &Resolution{
			Detail: "The User has no email",
		}, nil
	}

	externalUsr, err := opts.Resolver.GetExternalUserByEmail(ctx, usr.Spec.Email)
	if err != nil {
		return nil, err
	}

	if externalUsr == nil || externalUsr.ID == "" {
		return &Resolution{
			Detail: fmt.Sprintf("The provider has no external User with the email %s", usr.Spec.Email),
		}, nil
	}

	identity, err = setDiscoveredIdentity(ctx, opts.OcteliumC, opts.Integration, usr, externalUsr)
	if err != nil {
		return nil, err
	}

	return &Resolution{
		IsResolved: true,
		User:       usr,
		ExternalID: externalUsr.ID,
		Identity:   identity,
		Source:     accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
	}, nil
}

func getIdentityByExternalID(ctx context.Context, octeliumC octeliumc.ClientInterface,
	integration *accessv1.Integration, externalID string) (*accessv1.IntegrationIdentity, error) {
	itemList, err := octeliumC.AccessC().ListIntegrationIdentity(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.integrationRef.uid", integration.Metadata.Uid),
			urscsrv.FilterFieldEQValStr("status.externalID", externalID),
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if len(itemList.Items) != 1 {
		return nil, nil
	}

	return itemList.Items[0], nil
}

func getIdentityByUserUID(ctx context.Context, octeliumC octeliumc.ClientInterface,
	integration *accessv1.Integration, userUID string) (*accessv1.IntegrationIdentity, error) {
	itemList, err := octeliumC.AccessC().ListIntegrationIdentity(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("status.integrationRef.uid", integration.Metadata.Uid),
			urscsrv.FilterFieldEQValStr("status.userRef.uid", userUID),
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if len(itemList.Items) != 1 {
		return nil, nil
	}

	return itemList.Items[0], nil
}

func getUserByEmail(ctx context.Context, octeliumC octeliumc.ClientInterface,
	email string) (*corev1.User, error) {
	itemList, err := octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterFieldEQValStr("spec.email", email),
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if len(itemList.Items) != 1 {
		return nil, nil
	}

	return itemList.Items[0], nil
}

func setDiscoveredIdentity(ctx context.Context, octeliumC octeliumc.ClientInterface,
	integration *accessv1.Integration, usr *corev1.User,
	externalUsr *ExternalUser) (*accessv1.IntegrationIdentity, error) {
	item := &accessv1.IntegrationIdentity{
		Metadata: &metav1.Metadata{
			Name:     GenerateIdentityName(integration, externalUsr.ID),
			IsSystem: true,
		},
		Spec: &accessv1.IntegrationIdentity_Spec{},
		Status: &accessv1.IntegrationIdentity_Status{
			IntegrationRef:   umetav1.GetObjectReference(integration),
			UserRef:          umetav1.GetObjectReference(usr),
			ExternalID:       externalUsr.ID,
			Source:           accessv1.IntegrationIdentity_Status_EMAIL_DISCOVERY,
			VerifiedAt:       pbutils.Now(),
			ExternalUsername: externalUsr.Username,
			ExternalEmail:    externalUsr.Email,
		},
	}

	item, err := octeliumC.AccessC().CreateIntegrationIdentity(ctx, item)
	if err != nil {
		if !grpcerr.AlreadyExists(err) {
			return nil, grpcutils.InternalWithErr(err)
		}

		return getIdentityByExternalID(ctx, octeliumC, integration, externalUsr.ID)
	}

	return item, nil
}

func GenerateIdentityName(integration *accessv1.Integration, externalID string) string {
	return resourceNameFromKey("i", integration.Metadata.Uid, externalID)
}

func isEmailDiscoveryEnabled(integration *accessv1.Integration) bool {
	if integration.Spec == nil || integration.Spec.IdentityResolution == nil {
		return true
	}

	return !integration.Spec.IdentityResolution.DisableEmailDiscovery
}
