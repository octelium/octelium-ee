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
	"slices"

	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func IsAudienceRestricted(catalog *accessv1.Catalog) bool {
	audience := catalog.GetSpec().GetAudience()
	if audience == nil {
		return false
	}

	return len(audience.Users) > 0 || len(audience.Groups) > 0 || audience.Match != ""
}

func IsInAudience(ctx context.Context, celEngine *celengine.CELEngine,
	catalog *accessv1.Catalog, usr *corev1.User) (bool, error) {
	if usr == nil {
		return false, nil
	}

	if !IsAudienceRestricted(catalog) {
		return true, nil
	}

	audience := catalog.Spec.Audience

	if matchesUsersGroups(usr, audience.Users, audience.Groups) {
		return true, nil
	}

	if audience.Match == "" || celEngine == nil {
		return false, nil
	}

	return celEngine.EvalPolicy(ctx, audience.Match, map[string]any{
		"ctx": map[string]any{
			"user": pbutils.MustConvertToMap(usr),
		},
	})
}

func CanRequestOnBehalf(catalog *accessv1.Catalog, usr *corev1.User) bool {
	onBehalf := catalog.GetSpec().GetAudience().GetOnBehalf()
	if onBehalf == nil || usr == nil {
		return false
	}

	return matchesUsersGroups(usr, onBehalf.Users, onBehalf.Groups)
}

func IsCatalogOwner(catalog *accessv1.Catalog, usr *corev1.User) bool {
	owners := catalog.GetSpec().GetOwners()
	if owners == nil || usr == nil {
		return false
	}

	return matchesUsersGroups(usr, owners.Users, owners.Groups)
}

func matchesUsersGroups(usr *corev1.User, users, groups []string) bool {
	if usr == nil || usr.Metadata == nil {
		return false
	}

	if slices.Contains(users, usr.Metadata.Name) {
		return true
	}

	if usr.Spec == nil {
		return false
	}

	for _, grp := range usr.Spec.Groups {
		if slices.Contains(groups, grp) {
			return true
		}
	}

	return false
}

type CheckRequestableOpts struct {
	CELEngine *celengine.CELEngine
	Catalog   *accessv1.Catalog
	Requester *corev1.User
	Subject   *corev1.User
}

func CheckRequestable(ctx context.Context, o *CheckRequestableOpts) (*Problem, error) {
	catalog := o.Catalog

	if catalog == nil || catalog.Spec == nil {
		return newFatalProblem("spec.resource.catalog.catalogRef", "The Catalog does not exist"), nil
	}

	if catalog.Spec.IsDisabled {
		return newFatalProblem("spec.resource.catalog.catalogRef",
			"The Catalog %s is disabled", catalog.Metadata.Name), nil
	}

	if len(catalog.Spec.AccessLevels) == 0 {
		return newFatalProblem("spec.resource.catalog.catalogRef",
			"The Catalog %s has no access level", catalog.Metadata.Name), nil
	}

	isOnBehalf := o.Requester != nil && o.Subject != nil &&
		o.Requester.Metadata.Uid != o.Subject.Metadata.Uid

	if isOnBehalf && !CanRequestOnBehalf(catalog, o.Requester) {
		return newFatalProblem("spec.subject",
			"You are not allowed to request the Catalog %s on behalf of others", catalog.Metadata.Name), nil
	}

	ok, err := IsInAudience(ctx, o.CELEngine, catalog, o.Subject)
	if err != nil {
		return nil, err
	}

	if !ok {
		if isOnBehalf {
			return newFatalProblem("spec.subject",
				"The subject is not allowed to request the Catalog %s", catalog.Metadata.Name), nil
		}

		return newFatalProblem("spec.resource.catalog.catalogRef",
			"You are not allowed to request the Catalog %s", catalog.Metadata.Name), nil
	}

	return nil, nil
}
