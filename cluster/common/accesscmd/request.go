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
	"github.com/octelium/octelium/cluster/common/celengine"
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
	CELEngine *celengine.CELEngine

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

	prepared, err := PrepareRequest(ctx, &PrepareRequestOpts{
		OcteliumC:  opts.OcteliumC,
		CELEngine:  opts.CELEngine,
		Requester:  opts.Requester,
		Spec:       opts.Spec,
		ForSubject: opts.ForSubject,
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if err := ProblemsToError(prepared.Problems); err != nil {
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
		Spec: prepared.Spec,
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(opts.Requester),
			State: &accessv1.Request_Status_State{
				CreatedAt: pbutils.Now(),
				Status:    accessv1.Request_Status_State_PENDING,
				Reason:    accessv1.Request_Status_State_SUBMITTED,
			},
			Origin:        pbutils.Clone(opts.Origin).(*accessv1.Origin),
			ResolvedScope: prepared.Scope.Status,
			SpecRevision:  1,
			RequesterInfo: GetUserInfo(opts.Requester),
			SubjectInfo:   GetUserInfo(prepared.Subject),
		},
	}

	item, err = opts.OcteliumC.AccessC().CreateRequest(ctx, item)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return item, nil
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
