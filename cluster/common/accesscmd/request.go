// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accesscmd

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
)

const MaxPendingRequestsPerUser = 32

type CreateRequestOpts struct {
	OcteliumC octeliumc.ClientInterface

	Requester *corev1.User
	Spec      *accessv1.Request_Spec

	ForSubject bool

	Origin *accessv1.Origin
}

func CreateRequest(ctx context.Context, opts *CreateRequestOpts) (*accessv1.Request, error) {
	if opts == nil || opts.OcteliumC == nil || opts.Requester == nil {
		return nil, grpcutils.InvalidArg("Nil CreateRequest arguments")
	}

	if opts.Spec == nil {
		return nil, grpcutils.InvalidArg("Nil Spec")
	}

	if !IsUserEligible(opts.Requester) {
		return nil, grpcutils.Unauthorized("You are not allowed to create access Requests")
	}

	spec := pbutils.Clone(opts.Spec).(*accessv1.Request_Spec)

	if opts.ForSubject {
		if spec.Subject == nil || spec.Subject.GetUserRef() == nil {
			return nil, grpcutils.InvalidArg("Subject UserRef must be set")
		}
	} else {
		spec.Subject = &accessv1.Request_Spec_Subject{
			Type: &accessv1.Request_Spec_Subject_UserRef{
				UserRef: umetav1.GetObjectReference(opts.Requester),
			},
		}
	}

	if err := ValidateRequestSpec(ctx, opts.OcteliumC, spec); err != nil {
		return nil, err
	}

	if err := checkPendingRequestQuota(ctx, opts.OcteliumC, opts.Requester.Metadata.Uid); err != nil {
		return nil, err
	}

	name, err := generateRequestName(ctx, opts.OcteliumC)
	if err != nil {
		return nil, err
	}

	item := &accessv1.Request{
		Metadata: &metav1.Metadata{
			Name: name,
		},
		Spec: spec,
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(opts.Requester),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
			},
			Origin: pbutils.Clone(opts.Origin).(*accessv1.Origin),
		},
	}

	item, err = opts.OcteliumC.AccessC().CreateRequest(ctx, item)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
}

func ValidateRequestSpec(ctx context.Context, octeliumC octeliumc.ClientInterface,
	spec *accessv1.Request_Spec) error {
	if spec == nil {
		return grpcutils.InvalidArg("Nil Spec")
	}

	if spec.Resource == nil {
		return grpcutils.InvalidArg("Resource must be set")
	}

	switch spec.Resource.Type.(type) {
	case *accessv1.Request_Spec_Resource_ServiceRef:
		if err := apivalidation.CheckObjectRef(spec.Resource.GetServiceRef(),
			&apivalidation.CheckGetOptionsOpts{
				ParentsMax: 1,
			}); err != nil {
			return err
		}

		if _, err := octeliumC.CoreC().GetService(ctx,
			apivalidation.ObjectReferenceToRGetOptions(spec.Resource.GetServiceRef())); err != nil {
			if grpcerr.IsNotFound(err) {
				return grpcutils.NotFound("The Service does not exist")
			}
			return grpcutils.InternalWithErr(err)
		}

	case *accessv1.Request_Spec_Resource_Catalog_:
		if spec.Resource.GetCatalog() == nil {
			return grpcutils.InvalidArg("Catalog resource must be set")
		}

		if err := apivalidation.CheckObjectRef(spec.Resource.GetCatalog().GetCatalogRef(),
			&apivalidation.CheckGetOptionsOpts{}); err != nil {
			return err
		}

		if _, err := octeliumC.AccessC().GetCatalog(ctx,
			apivalidation.ObjectReferenceToRGetOptions(spec.Resource.GetCatalog().GetCatalogRef())); err != nil {
			if grpcerr.IsNotFound(err) {
				return grpcutils.NotFound("The Catalog does not exist")
			}
			return grpcutils.InternalWithErr(err)
		}

	default:
		return grpcutils.InvalidArg("Resource type must be set")
	}

	if spec.Subject != nil {
		switch spec.Subject.Type.(type) {
		case *accessv1.Request_Spec_Subject_UserRef:
			if err := apivalidation.CheckObjectRef(spec.Subject.GetUserRef(),
				&apivalidation.CheckGetOptionsOpts{}); err != nil {
				return err
			}

			if _, err := octeliumC.CoreC().GetUser(ctx,
				apivalidation.ObjectReferenceToRGetOptions(spec.Subject.GetUserRef())); err != nil {
				if grpcerr.IsNotFound(err) {
					return grpcutils.NotFound("The User does not exist")
				}
				return grpcutils.InternalWithErr(err)
			}

		default:
			return grpcutils.InvalidArg("Subject type must be set")
		}
	}

	switch spec.Urgency {
	case accessv1.Request_Spec_URGENCY_UNSET,
		accessv1.Request_Spec_VERY_LOW,
		accessv1.Request_Spec_LOW,
		accessv1.Request_Spec_NORMAL,
		accessv1.Request_Spec_HIGH,
		accessv1.Request_Spec_VERY_HIGH,
		accessv1.Request_Spec_HIGHEST:
	default:
		return grpcutils.InvalidArg("Invalid Urgency")
	}

	if len(spec.Justification) > MaxJustificationLen {
		return grpcutils.InvalidArg("Justification is too long")
	}

	if spec.Duration != nil {
		if err := apivalidation.ValidateDuration(spec.Duration); err != nil {
			return err
		}
	}

	return nil
}

func checkPendingRequestQuota(ctx context.Context, octeliumC octeliumc.ClientInterface,
	requesterUID string) error {
	itemList, err := octeliumC.AccessC().ListRequest(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(requesterUID),
			urscsrv.FilterFieldEQValStr("status.state.status", "PENDING"),
		},
		Paginate:     true,
		ItemsPerPage: MaxPendingRequestsPerUser + 1,
	})
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	if len(itemList.Items) >= MaxPendingRequestsPerUser {
		return grpcutils.InvalidArg("You have too many pending access Requests")
	}

	return nil
}

func generateRequestName(ctx context.Context, octeliumC octeliumc.ClientInterface) (string, error) {
	const attemptsPerLength = 32

	for n := 3; n <= 8; n++ {
		for i := 0; i < attemptsPerLength; i++ {
			name := utilrand.GetRandomStringCanonical(n)

			_, err := octeliumC.AccessC().GetRequest(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			if err == nil {
				continue
			}

			if grpcerr.IsNotFound(err) {
				return name, nil
			}

			return "", grpcutils.InternalWithErr(err)
		}
	}

	return "", grpcutils.Internal("Could not generate a unique Request name")
}
