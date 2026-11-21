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
	"time"

	"github.com/octelium/octelium-ee/cluster/common/accesscmd"
	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/serr"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

type previewResult struct {
	preview *accessv1.RequestPreview
	request *accessv1.Request
	eval    *accesscmd.EvalResult
}

func buildPreview(ctx context.Context, octeliumC octeliumc.ClientInterface,
	celEngine *celengine.CELEngine, prepared *accesscmd.PreparedRequest, collectAll bool) (*previewResult, error) {
	ret := &previewResult{
		preview: &accessv1.RequestPreview{
			Problems: accesscmd.ProblemsToPreview(prepared.Problems),
		},
	}

	preview := ret.preview

	if prepared.Catalog != nil {
		preview.Requirements = prepared.Catalog.GetSpec().GetRequirements()
	}

	for _, catalog := range prepared.Candidates {
		itm, err := toUserCatalog(ctx, octeliumC, catalog, nil)
		if err != nil {
			return nil, err
		}
		preview.CandidateCatalogs = append(preview.CandidateCatalogs, itm)
	}

	if !prepared.IsRequestable() {
		preview.Outcome = accessv1.RequestPreview_NOT_REQUESTABLE
		for _, p := range prepared.Problems {
			if p.Fatal {
				preview.Message = p.Message
				break
			}
		}
		return ret, nil
	}

	req := &accessv1.Request{
		Spec: prepared.Spec,
		Status: &accessv1.Request_Status{
			UserRef:       umetav1.GetObjectReference(prepared.Requester),
			ResolvedScope: prepared.Scope.Status,
			SpecRevision:  1,
			RequesterInfo: accesscmd.GetUserInfo(prepared.Requester),
			SubjectInfo:   accesscmd.GetUserInfo(prepared.Subject),
		},
	}
	ret.request = req

	preview.Request = accesscmd.RedactRequest(req)
	preview.ResolvedScope = preview.Request.Status.ResolvedScope
	preview.MaxDuration = accesscmd.DurationFromGo(accesscmd.GetMaxDuration(prepared.Scope.Level, nil))
	preview.EffectiveDuration = accesscmd.DurationFromGo(
		accesscmd.GetEffectiveDuration(prepared.Spec, prepared.Scope.Level, nil))

	existing, err := getExistingRequests(ctx, octeliumC, prepared.Subject, prepared.Spec)
	if err != nil {
		return nil, err
	}
	preview.ExistingRequests = existing

	res, err := accesscmd.NewEvaluator(octeliumC, celEngine).Evaluate(ctx, &accesscmd.EvalInput{
		Request:   req,
		Requester: prepared.Requester,
		Subject:   prepared.Subject,
		Scope:     prepared.Scope,
	}, collectAll)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}
	ret.eval = res

	if res.Rule == nil {
		preview.Outcome = accessv1.RequestPreview_NO_MATCH
		preview.Message = prepared.ClusterConfig.GetSpec().GetRequest().GetNoMatchMessage()
		return ret, nil
	}

	workflow, err := accesscmd.BuildWorkflow(ctx, octeliumC, res.Rule)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	preview.Workflow = workflow
	preview.Message = res.Rule.Message
	preview.MaxDuration = accesscmd.DurationFromGo(accesscmd.GetMaxDuration(prepared.Scope.Level, res.Rule))
	preview.EffectiveDuration = accesscmd.DurationFromGo(
		accesscmd.GetEffectiveDuration(prepared.Spec, prepared.Scope.Level, res.Rule))

	req.Status.PolicyRef = umetav1.GetObjectReference(res.Policy)
	req.Status.Rule = pbutils.Clone(res.Rule).(*accessv1.Policy_Spec_Rule)
	req.Status.Workflow = workflow
	req.Status.EffectiveDuration = preview.EffectiveDuration

	switch res.Rule.Effect {
	case accessv1.Policy_Spec_Rule_DENY:
		preview.Outcome = accessv1.RequestPreview_DENIED
		return ret, nil
	case accessv1.Policy_Spec_Rule_AUTO_APPROVE:
		preview.Outcome = accessv1.RequestPreview_AUTO_APPROVED
	default:
		preview.Outcome = accessv1.RequestPreview_REVIEW
	}

	if len(prepared.Problems) > 0 {
		preview.Outcome = accessv1.RequestPreview_INCOMPLETE
	}

	return ret, nil
}

func getExistingRequests(ctx context.Context, octeliumC octeliumc.ClientInterface,
	subject *corev1.User, spec *accessv1.Request_Spec) ([]*accessv1.RequestPreview_ExistingRequest, error) {
	resCatalog := spec.GetResource().GetCatalog()
	if subject == nil || resCatalog == nil || resCatalog.CatalogRef == nil {
		return nil, nil
	}

	itemList, err := listAllRequests(ctx, octeliumC,
		urscsrv.FilterFieldEQValStr("spec.subject.userRef.uid", subject.Metadata.Uid),
		urscsrv.FilterFieldEQValStr("spec.resource.catalog.catalogRef.uid", resCatalog.CatalogRef.Uid))
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	now := time.Now()
	var ret []*accessv1.RequestPreview_ExistingRequest

	for _, itm := range itemList {
		if itm.Spec.GetResource().GetCatalog().GetAccessLevel() != resCatalog.AccessLevel {
			continue
		}

		if !accesscmd.IsPending(itm) && !accesscmd.IsAccessActive(itm, now) {
			continue
		}

		ret = append(ret, &accessv1.RequestPreview_ExistingRequest{
			RequestRef:   umetav1.GetObjectReference(itm),
			State:        itm.Status.State.Status,
			AccessEndsAt: itm.Status.AccessEndsAt,
		})
	}

	return ret, nil
}

func toUserCatalog(ctx context.Context, octeliumC octeliumc.ClientInterface,
	catalog *accessv1.Catalog, viewer *accessv1.UserCatalog_Viewer) (*accessv1.UserCatalog, error) {
	ret := &accessv1.UserCatalog{
		Metadata:             userCatalogMetadata(catalog),
		Requirements:         catalog.GetSpec().GetRequirements(),
		AllowPartialRequests: catalog.GetSpec().GetAllowPartialRequests(),
		TotalService:         catalog.GetStatus().GetTotalService(),
		Viewer:               viewer,
		Membership:           accesscmd.GetMembership(catalog),
	}

	cc, err := accesscmd.GetClusterConfig(ctx, octeliumC)
	if err != nil {
		return nil, serr.InternalWithErr(err)
	}

	for _, level := range catalog.GetSpec().GetAccessLevels() {
		itm := &accessv1.UserCatalog_AccessLevel{
			Name:            level.Name,
			DisplayName:     level.DisplayName,
			Description:     level.Description,
			DefaultDuration: level.DefaultDuration,
			MaxDuration:     accesscmd.DurationFromGo(accesscmd.GetMaxDuration(level, nil)),
			MinAAL:          level.GetVerification().GetMinAAL(),
		}

		if accesscmd.IsFullAccessLevel(level) {
			itm.Type = accessv1.UserCatalog_AccessLevel_FULL
		} else {
			itm.Type = accessv1.UserCatalog_AccessLevel_RESTRICTED
		}

		itm.OverridesGuardrails = accesscmd.GetAccessLevelPriority(level,
			accesscmd.GetDefaultAccessPriority(cc)) < 0

		ret.AccessLevels = append(ret.AccessLevels, itm)
	}

	if owners := catalog.GetSpec().GetOwners(); owners != nil {
		for _, name := range owners.Users {
			itm := &accessv1.UserCatalog_Owner{
				Type:        accessv1.UserCatalog_Owner_USER,
				Name:        name,
				DisplayName: name,
			}

			usr, err := octeliumC.CoreC().GetUser(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			if err == nil {
				itm.DisplayName = accesscmd.GetUserDisplayName(usr)
			}

			ret.Owners = append(ret.Owners, itm)
		}

		for _, name := range owners.Groups {
			itm := &accessv1.UserCatalog_Owner{
				Type:        accessv1.UserCatalog_Owner_GROUP,
				Name:        name,
				DisplayName: name,
			}

			grp, err := octeliumC.CoreC().GetGroup(ctx, &rmetav1.GetOptions{
				Name: name,
			})
			if err == nil && grp.Metadata.DisplayName != "" {
				itm.DisplayName = grp.Metadata.DisplayName
			}

			ret.Owners = append(ret.Owners, itm)
		}
	}

	return ret, nil
}

func userCatalogMetadata(catalog *accessv1.Catalog) *metav1.Metadata {
	md := catalog.GetMetadata()
	if md == nil {
		return nil
	}

	return &metav1.Metadata{
		Uid:         md.Uid,
		Name:        md.Name,
		DisplayName: md.DisplayName,
		Description: md.Description,
		Labels:      md.Labels,
		Tags:        md.Tags,
		PicURL:      md.PicURL,
		CreatedAt:   md.CreatedAt,
		UpdatedAt:   md.UpdatedAt,
	}
}
