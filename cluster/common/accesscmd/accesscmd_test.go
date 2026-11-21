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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/accessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/celengine"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accessTest struct {
	ctx       context.Context
	octeliumC octeliumc.ClientInterface
	celEngine *celengine.CELEngine
}

func newAccessTest(t *testing.T) *accessTest {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	celEngine, err := celengine.New(ctx, nil)
	require.NoError(t, err)

	return &accessTest{
		ctx:       ctx,
		octeliumC: tst.C.OcteliumC,
		celEngine: celEngine,
	}
}

func (a *accessTest) createUser(t *testing.T, groups ...string) *corev1.User {
	usr, err := a.octeliumC.CoreC().CreateUser(a.ctx, &corev1.User{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &corev1.User_Spec{
			Type:   corev1.User_Spec_HUMAN,
			Groups: groups,
		},
		Status: &corev1.User_Status{},
	})
	require.NoError(t, err)
	return usr
}

func (a *accessTest) createService(t *testing.T, md *metav1.Metadata) *corev1.Service {
	if md == nil {
		md = &metav1.Metadata{}
	}
	md.Name = fmt.Sprintf("%s.default", utilrand.GetRandomStringCanonical(8))

	svc, err := a.octeliumC.CoreC().CreateService(a.ctx, &corev1.Service{
		Metadata: md,
		Spec:     &corev1.Service_Spec{},
		Status:   &corev1.Service_Status{},
	})
	require.NoError(t, err)
	return svc
}

func fullLevel(name string) *accessv1.Catalog_Spec_AccessLevel {
	return &accessv1.Catalog_Spec_AccessLevel{
		Name: name,
		Type: &accessv1.Catalog_Spec_AccessLevel_Full_{
			Full: &accessv1.Catalog_Spec_AccessLevel_Full{},
		},
	}
}

func (a *accessTest) createCatalog(t *testing.T, fn func(spec *accessv1.Catalog_Spec),
	services ...*corev1.Service) *accessv1.Catalog {
	var names []string
	for _, svc := range services {
		names = append(names, svc.Metadata.Name)
	}

	spec := &accessv1.Catalog_Spec{
		ResourceCollection: &accessv1.Catalog_Spec_ResourceCollection{
			Service: &accessv1.Catalog_Spec_ResourceCollection_Service{
				Services: names,
			},
		},
		AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{fullLevel("read"), fullLevel("write")},
	}

	if fn != nil {
		fn(spec)
	}

	catalog, err := a.octeliumC.AccessC().CreateCatalog(a.ctx, &accessv1.Catalog{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     spec,
		Status:   &accessv1.Catalog_Status{},
	})
	require.NoError(t, err)
	return catalog
}

func catalogSpec(ref *metav1.ObjectReference) *accessv1.Request_Spec {
	return &accessv1.Request_Spec{
		Resource: &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_Catalog_{
				Catalog: &accessv1.Request_Spec_Resource_Catalog{
					CatalogRef: ref,
				},
			},
		},
	}
}

func (a *accessTest) prepare(t *testing.T, requester *corev1.User,
	spec *accessv1.Request_Spec, forSubject bool) *PreparedRequest {
	ret, err := PrepareRequest(a.ctx, &PrepareRequestOpts{
		OcteliumC:  a.octeliumC,
		CELEngine:  a.celEngine,
		Requester:  requester,
		Spec:       spec,
		ForSubject: forSubject,
	})
	require.NoError(t, err)
	return ret
}

func problemMessages(problems []*Problem) string {
	var ret []string
	for _, p := range problems {
		ret = append(ret, p.Message)
	}
	return strings.Join(ret, "; ")
}

func TestPrepareRequestCanonicalizesReferences(t *testing.T) {
	a := newAccessTest(t)

	requester := a.createUser(t)
	subject := a.createUser(t)
	svc := a.createService(t, nil)
	catalog := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Audience = &accessv1.Catalog_Spec_Audience{
			OnBehalf: &accessv1.Catalog_Spec_Audience_OnBehalf{Users: []string{requester.Metadata.Name}},
		}
	}, svc)

	spec := catalogSpec(&metav1.ObjectReference{Name: catalog.Metadata.Name})
	spec.Subject = &accessv1.Request_Spec_Subject{
		Type: &accessv1.Request_Spec_Subject_UserRef{
			UserRef: &metav1.ObjectReference{Name: subject.Metadata.Name},
		},
	}

	prepared := a.prepare(t, requester, spec, true)
	require.Empty(t, prepared.Problems, problemMessages(prepared.Problems))

	assert.Equal(t, catalog.Metadata.Uid, prepared.Spec.Resource.GetCatalog().CatalogRef.Uid)
	assert.Empty(t, prepared.Spec.Resource.GetCatalog().CatalogRef.ResourceVersion)
	assert.Equal(t, "read", prepared.Spec.Resource.GetCatalog().AccessLevel)
	assert.Equal(t, subject.Metadata.Uid, prepared.Spec.Subject.GetUserRef().Uid)
	assert.NotNil(t, prepared.Spec.Deadline)
	assert.Equal(t, svc.Metadata.Uid, prepared.Scope.Status.ServiceRefs[0].Uid)

	noOnBehalf := a.prepare(t, a.createUser(t), spec, true)
	require.True(t, HasFatalProblem(noOnBehalf.Problems))
	assert.Contains(t, problemMessages(noOnBehalf.Problems), "on behalf of others")
}

func TestPrepareRequestServiceShorthand(t *testing.T) {
	a := newAccessTest(t)

	requester := a.createUser(t)
	svc := a.createService(t, nil)

	spec := &accessv1.Request_Spec{
		Resource: &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_ServiceRef{
				ServiceRef: &metav1.ObjectReference{Name: svc.Metadata.Name},
			},
		},
	}

	prepared := a.prepare(t, requester, spec, false)
	require.True(t, HasFatalProblem(prepared.Problems))
	assert.Contains(t, problemMessages(prepared.Problems), "is not part of any Catalog")

	first := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.AllowPartialRequests = true
	}, svc, a.createService(t, nil))

	prepared = a.prepare(t, requester, spec, false)
	require.Empty(t, prepared.Problems, problemMessages(prepared.Problems))
	resCatalog := prepared.Spec.Resource.GetCatalog()
	require.NotNil(t, resCatalog)
	assert.Equal(t, first.Metadata.Uid, resCatalog.CatalogRef.Uid)
	require.Len(t, resCatalog.ServiceRefs, 1)
	assert.Equal(t, svc.Metadata.Uid, resCatalog.ServiceRefs[0].Uid)
	require.Len(t, prepared.Scope.Status.ServiceRefs, 1)

	whole := a.createCatalog(t, nil, svc)

	prepared = a.prepare(t, requester, spec, false)
	require.True(t, HasFatalProblem(prepared.Problems))
	assert.Contains(t, problemMessages(prepared.Problems), "several Catalogs")
	assert.Len(t, prepared.Candidates, 2)

	a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Audience = &accessv1.Catalog_Spec_Audience{Users: []string{"someone-else"}}
	}, svc)
	prepared = a.prepare(t, requester, spec, false)
	assert.Len(t, prepared.Candidates, 2)

	first.Spec.AllowPartialRequests = false
	_, err := a.octeliumC.AccessC().UpdateCatalog(a.ctx, first)
	require.NoError(t, err)
	_, err = a.octeliumC.AccessC().DeleteCatalog(a.ctx, &rmetav1.DeleteOptions{Uid: whole.Metadata.Uid})
	require.NoError(t, err)

	prepared = a.prepare(t, requester, spec, false)
	require.True(t, HasFatalProblem(prepared.Problems))
	assert.Contains(t, problemMessages(prepared.Problems), "Request the Catalog instead")
	assert.Nil(t, prepared.Scope)

	only := a.createService(t, nil)
	whole = a.createCatalog(t, nil, only)

	prepared = a.prepare(t, requester, &accessv1.Request_Spec{
		Resource: &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(only),
			},
		},
	}, false)
	require.Empty(t, prepared.Problems, problemMessages(prepared.Problems))
	assert.Equal(t, whole.Metadata.Uid, prepared.Spec.Resource.GetCatalog().CatalogRef.Uid)
	assert.Empty(t, prepared.Spec.Resource.GetCatalog().ServiceRefs)
	require.Len(t, prepared.Scope.Status.ServiceRefs, 1)
}

func TestPrepareRequestExcludesHiddenServices(t *testing.T) {
	a := newAccessTest(t)

	requester := a.createUser(t)
	hidden := a.createService(t, &metav1.Metadata{IsSystem: true, IsUserHidden: true})
	a.createCatalog(t, nil, hidden)

	prepared := a.prepare(t, requester, &accessv1.Request_Spec{
		Resource: &accessv1.Request_Spec_Resource{
			Type: &accessv1.Request_Spec_Resource_ServiceRef{
				ServiceRef: umetav1.GetObjectReference(hidden),
			},
		},
	}, false)
	require.True(t, HasFatalProblem(prepared.Problems))

	visible := a.createService(t, nil)
	catalog := a.createCatalog(t, nil, visible, hidden)

	prepared = a.prepare(t, requester, catalogSpec(umetav1.GetObjectReference(catalog)), false)
	require.Empty(t, prepared.Problems, problemMessages(prepared.Problems))
	require.Len(t, prepared.Scope.Status.ServiceRefs, 1)
	assert.Equal(t, visible.Metadata.Uid, prepared.Scope.Status.ServiceRefs[0].Uid)
	assert.Equal(t, uint32(1), prepared.Scope.Status.TotalExcludedService)
}

func TestPrepareRequestRequirementsAndFields(t *testing.T) {
	a := newAccessTest(t)

	requester := a.createUser(t)
	catalog := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Requirements = &accessv1.Catalog_Spec_Requirements{
			Justification: &accessv1.Catalog_Spec_Requirements_Justification{
				IsRequired: true,
				MinLength:  10,
			},
			Ticket: &accessv1.Catalog_Spec_Requirements_Ticket{
				IsRequired: true,
				Pattern:    `^INC-[0-9]+$`,
				Label:      "incident",
			},
		}
	}, a.createService(t, nil))

	spec := catalogSpec(umetav1.GetObjectReference(catalog))
	prepared := a.prepare(t, requester, spec, false)
	assert.False(t, HasFatalProblem(prepared.Problems))
	assert.Contains(t, problemMessages(prepared.Problems), "A justification is required")
	assert.Contains(t, problemMessages(prepared.Problems), "A incident is required")

	spec.Justification = "short"
	spec.Ticket = "INC-12x"
	prepared = a.prepare(t, requester, spec, false)
	assert.Contains(t, problemMessages(prepared.Problems), "at least 10 characters")
	assert.Contains(t, problemMessages(prepared.Problems), "invalid format")

	spec.Justification = strings.Repeat("é", MaxJustificationLen)
	spec.Ticket = "INC-42"
	prepared = a.prepare(t, requester, spec, false)
	assert.Empty(t, prepared.Problems, problemMessages(prepared.Problems))

	spec.Justification = strings.Repeat("é", MaxJustificationLen+1)
	prepared = a.prepare(t, requester, spec, false)
	assert.True(t, HasFatalProblem(prepared.Problems))

	spec.Justification = "investigating the incident"
	spec.StartAt = pbutils.Timestamp(time.Now().Add(DefaultMaxStartDelay + time.Hour))
	prepared = a.prepare(t, requester, spec, false)
	assert.Contains(t, problemMessages(prepared.Problems), "cannot start later")

	spec.StartAt = nil
	spec.Deadline = pbutils.Timestamp(time.Now().Add(-time.Minute))
	prepared = a.prepare(t, requester, spec, false)
	assert.Contains(t, problemMessages(prepared.Problems), "deadline must be in the future")

	spec.Deadline = nil
	spec.Resource.GetCatalog().AccessLevel = "missing"
	prepared = a.prepare(t, requester, spec, false)
	assert.True(t, HasFatalProblem(prepared.Problems))
	assert.Contains(t, problemMessages(prepared.Problems), "has no access level missing")

	err := ProblemsToError(prepared.Problems)
	assert.True(t, grpcerr.IsInvalidArg(err))
}

func TestPrepareRequestAudience(t *testing.T) {
	a := newAccessTest(t)

	grp, err := a.octeliumC.CoreC().CreateGroup(a.ctx, &corev1.Group{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &corev1.Group_Spec{},
		Status:   &corev1.Group_Status{},
	})
	require.NoError(t, err)

	member := a.createUser(t, grp.Metadata.Name)
	named := a.createUser(t)
	other := a.createUser(t)

	catalog := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Audience = &accessv1.Catalog_Spec_Audience{
			Users:  []string{named.Metadata.Name},
			Groups: []string{grp.Metadata.Name},
		}
	}, a.createService(t, nil))

	for _, tc := range []struct {
		usr  *corev1.User
		want bool
	}{{member, true}, {named, true}, {other, false}} {
		prepared := a.prepare(t, tc.usr, catalogSpec(umetav1.GetObjectReference(catalog)), false)
		assert.Equal(t, tc.want, prepared.IsRequestable(), problemMessages(prepared.Problems))
	}

	celCatalog := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Audience = &accessv1.Catalog_Spec_Audience{
			Match: fmt.Sprintf(`ctx.user.metadata.name == "%s"`, other.Metadata.Name),
		}
	}, a.createService(t, nil))

	assert.True(t, a.prepare(t, other, catalogSpec(umetav1.GetObjectReference(celCatalog)), false).IsRequestable())
	assert.False(t, a.prepare(t, member, catalogSpec(umetav1.GetObjectReference(celCatalog)), false).IsRequestable())

	disabled := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.IsDisabled = true
	}, a.createService(t, nil))
	assert.False(t, a.prepare(t, member, catalogSpec(umetav1.GetObjectReference(disabled)), false).IsRequestable())

	owners := &accessv1.Catalog{Spec: &accessv1.Catalog_Spec{Owners: &accessv1.Catalog_Spec_Owners{Groups: []string{grp.Metadata.Name}}}}
	assert.True(t, IsCatalogOwner(owners, member))
	assert.False(t, IsCatalogOwner(owners, other))
	assert.False(t, IsAudienceRestricted(&accessv1.Catalog{Spec: &accessv1.Catalog_Spec{}}))
}

func TestPrepareRequestExtension(t *testing.T) {
	a := newAccessTest(t)

	requester := a.createUser(t)
	catalog := a.createCatalog(t, nil, a.createService(t, nil))

	extended, err := a.octeliumC.AccessC().CreateRequest(a.ctx, &accessv1.Request{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec: &accessv1.Request_Spec{
			Resource: &accessv1.Request_Spec_Resource{
				Type: &accessv1.Request_Spec_Resource_Catalog_{
					Catalog: &accessv1.Request_Spec_Resource_Catalog{
						CatalogRef:  umetav1.GetObjectReference(catalog),
						AccessLevel: "read",
					},
				},
			},
			Subject: &accessv1.Request_Spec_Subject{
				Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: umetav1.GetObjectReference(requester)},
			},
		},
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(requester),
			State: &accessv1.Request_Status_State{
				Status: accessv1.Request_Status_State_APPROVED,
			},
		},
	})
	require.NoError(t, err)

	spec := catalogSpec(umetav1.GetObjectReference(catalog))
	spec.ExtendsRef = &metav1.ObjectReference{Name: extended.Metadata.Name}

	prepared := a.prepare(t, requester, spec, false)
	require.Empty(t, prepared.Problems, problemMessages(prepared.Problems))
	assert.Equal(t, extended.Metadata.Uid, prepared.Spec.ExtendsRef.Uid)

	spec.Resource.GetCatalog().AccessLevel = "write"
	prepared = a.prepare(t, requester, spec, false)
	assert.Contains(t, problemMessages(prepared.Problems), "different access level")

	spec.Resource.GetCatalog().AccessLevel = ""
	prepared = a.prepare(t, a.createUser(t), spec, false)
	assert.Contains(t, problemMessages(prepared.Problems), "different subject")
}

func TestEffectiveDuration(t *testing.T) {
	hours := func(h uint32) *metav1.Duration {
		return &metav1.Duration{Type: &metav1.Duration_Hours{Hours: h}}
	}

	level := fullLevel("read")
	assert.Equal(t, DefaultMaxAccessDuration, GetEffectiveDuration(&accessv1.Request_Spec{}, level, nil))
	assert.Equal(t, DefaultMaxAccessDuration, GetEffectiveDuration(&accessv1.Request_Spec{Duration: hours(72)}, level, nil))
	assert.Equal(t, 3*time.Hour, GetEffectiveDuration(&accessv1.Request_Spec{Duration: hours(3)}, level, nil))

	level.MaxDuration = hours(96)
	level.DefaultDuration = hours(2)
	assert.Equal(t, 2*time.Hour, GetEffectiveDuration(&accessv1.Request_Spec{}, level, nil))
	assert.Equal(t, 72*time.Hour, GetEffectiveDuration(&accessv1.Request_Spec{Duration: hours(72)}, level, nil))

	rule := &accessv1.Policy_Spec_Rule{MaxDuration: hours(4)}
	assert.Equal(t, 4*time.Hour, GetEffectiveDuration(&accessv1.Request_Spec{Duration: hours(72)}, level, rule))
	assert.Equal(t, 4*time.Hour, GetMaxDuration(level, rule))

	rule.MaxDuration = hours(200)
	assert.Equal(t, 96*time.Hour, GetMaxDuration(level, rule))

	assert.Nil(t, DurationFromGo(0))
	assert.Equal(t, uint32(90), DurationFromGo(90*time.Second).GetSeconds())
}

func TestAccessLevelPriority(t *testing.T) {
	full := fullLevel("full")
	assert.Equal(t, int32(0), GetAccessLevelPriority(full, 0))
	assert.Equal(t, int32(-1), GetAccessLevelPriority(full, -1))
	assert.Equal(t, int32(-4), GetAccessLevelPriority(full, -10))

	explicit := int32(-2)
	full.GetFull().Priority = &explicit
	assert.Equal(t, int32(-2), GetAccessLevelPriority(full, 0))

	restricted := &accessv1.Catalog_Spec_AccessLevel{
		Type: &accessv1.Catalog_Spec_AccessLevel_Restricted_{
			Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{
				InlinePolicies: []*corev1.InlinePolicy{
					{Spec: &corev1.Policy_Spec{Rules: []*corev1.Policy_Spec_Rule{{Priority: 2}, {Priority: -3}}}},
				},
			},
		},
	}
	assert.Equal(t, int32(-3), GetAccessLevelPriority(restricted, -1))
	assert.False(t, IsFullAccessLevel(restricted))
	assert.True(t, IsFullAccessLevel(full))

	catalog := &accessv1.Catalog{Spec: &accessv1.Catalog_Spec{AccessLevels: []*accessv1.Catalog_Spec_AccessLevel{full, restricted}}}
	assert.Equal(t, full, GetAccessLevel(catalog, ""))
	assert.Nil(t, GetAccessLevel(catalog, "missing"))
	assert.Nil(t, GetAccessLevel(&accessv1.Catalog{Spec: &accessv1.Catalog_Spec{}}, ""))
}

func TestBuildPolicyTrigger(t *testing.T) {
	subject := &metav1.ObjectReference{Name: "alice", Uid: "uid-alice"}
	startsAt := time.Now().UTC().Truncate(time.Second)
	endsAt := startsAt.Add(time.Hour)

	newRequest := func(level *accessv1.Catalog_Spec_AccessLevel, priority int32) *accessv1.Request {
		return &accessv1.Request{
			Metadata: &metav1.Metadata{Name: "req", Uid: "uid-req"},
			Spec: &accessv1.Request_Spec{
				Subject: &accessv1.Request_Spec_Subject{
					Type: &accessv1.Request_Spec_Subject_UserRef{UserRef: subject},
				},
			},
			Status: &accessv1.Request_Status{
				AccessStartsAt: pbutils.Timestamp(startsAt),
				AccessEndsAt:   pbutils.Timestamp(endsAt),
				ResolvedScope: &accessv1.Request_Status_ResolvedScope{
					AccessLevel: level,
					Priority:    priority,
					ServiceRefs: []*metav1.ObjectReference{
						{Name: "a.default", Uid: "uid-a", ResourceVersion: "1"},
						{Name: "b.default", Uid: "uid-b"},
					},
					NamespaceRefs: []*metav1.ObjectReference{{Name: "ns", Uid: "uid-ns"}},
				},
			},
		}
	}

	pt, err := BuildPolicyTrigger(newRequest(fullLevel("full"), -2))
	require.NoError(t, err)
	assert.Equal(t, "access-request-uid-req", pt.Metadata.Name)
	assert.True(t, pt.Metadata.IsSystemHidden)
	require.Len(t, pt.Status.InlinePolicies, 1)
	assert.Equal(t, DefaultGrantInlinePolicyName, pt.Status.InlinePolicies[0].Name)
	assert.Equal(t, int32(-2), pt.Status.InlinePolicies[0].Spec.Rules[0].Priority)
	assert.Equal(t, corev1.Policy_Spec_Rule_ALLOW, pt.Status.InlinePolicies[0].Spec.Rules[0].Effect)

	all := pt.Status.PreCondition.GetAll()
	require.Len(t, all.Of, 4)
	assert.Equal(t, "uid-alice", all.Of[0].GetUserRef().Uid)
	require.Len(t, all.Of[1].GetAny().Of, 3)
	assert.Empty(t, all.Of[1].GetAny().Of[0].GetServiceRef().ResourceVersion)
	nsBranch := all.Of[1].GetAny().Of[2].GetAll()
	require.Len(t, nsBranch.Of, 2)
	assert.Equal(t, "uid-ns", nsBranch.Of[0].GetNamespaceRef().Uid)
	assert.Equal(t, nonSystemServiceMatch, nsBranch.Of[1].GetCondition().GetMatch())
	assert.Equal(t, startsAt, all.Of[2].GetNotBefore().AsTime())
	assert.Equal(t, endsAt, all.Of[3].GetNotAfter().AsTime())

	restricted := &accessv1.Catalog_Spec_AccessLevel{
		Type: &accessv1.Catalog_Spec_AccessLevel_Restricted_{
			Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{
				Policies: []string{"", "read-only"},
				InlinePolicies: []*corev1.InlinePolicy{
					nil,
					{Name: "get", Spec: &corev1.Policy_Spec{}},
				},
			},
		},
		Verification: &accessv1.Catalog_Spec_AccessLevel_Verification{
			MinAAL: corev1.Session_Status_Authentication_Info_AAL3,
		},
	}

	req := newRequest(restricted, 0)
	pt, err = BuildPolicyTrigger(req)
	require.NoError(t, err)
	assert.Equal(t, []string{"read-only"}, pt.Status.Policies)
	require.Len(t, pt.Status.InlinePolicies, 1)
	assert.Equal(t, "get", pt.Status.InlinePolicies[0].Name)

	all = pt.Status.PreCondition.GetAll()
	require.Len(t, all.Of, 5)
	assert.Equal(t, `ctx.session.status.authentication.info.aal in ["AAL3"]`, all.Of[4].GetCondition().GetMatch())

	_, err = BuildPolicyTrigger(newRequest(&accessv1.Catalog_Spec_AccessLevel{Name: "untyped"}, 0))
	assert.Error(t, err)

	req.Status.ResolvedScope.ServiceRefs = nil
	req.Status.ResolvedScope.NamespaceRefs = nil
	_, err = BuildPolicyTrigger(req)
	assert.Error(t, err)

	req.Status.ResolvedScope = nil
	_, err = BuildPolicyTrigger(req)
	assert.Error(t, err)
}

func TestPolicyTriggerNamespaceExcludesSystemServices(t *testing.T) {
	ctx := context.Background()
	celEngine, err := celengine.New(ctx, nil)
	require.NoError(t, err)

	cond := &corev1.Condition{
		Type: &corev1.Condition_Match{
			Match: nonSystemServiceMatch,
		},
	}

	for _, tc := range []struct {
		md   *metav1.Metadata
		want bool
	}{
		{&metav1.Metadata{Name: "app.production"}, true},
		{&metav1.Metadata{Name: "app.production", IsSystem: true}, false},
		{&metav1.Metadata{Name: "app.production", IsSystemHidden: true}, false},
		{&metav1.Metadata{Name: "app.production", IsUserHidden: true}, false},
	} {
		reqCtx, err := pbutils.ConvertToMap(&corev1.RequestContext{
			Service: &corev1.Service{Metadata: tc.md},
		})
		require.NoError(t, err)

		matched, err := celEngine.EvalCondition(ctx, cond, map[string]any{"ctx": reqCtx})
		require.NoError(t, err)
		assert.Equal(t, tc.want, matched, "%+v", tc.md)
	}

	matched, err := celEngine.EvalCondition(ctx, cond, map[string]any{"ctx": map[string]any{}})
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestSetRequestState(t *testing.T) {
	req := &accessv1.Request{
		Status: &accessv1.Request_Status{
			Review: &accessv1.Request_Status_Review{
				Steps: []*accessv1.Request_Status_Review_StepState{{Name: "first"}},
			},
			Grant: &accessv1.Request_Status_Grant{State: accessv1.Request_Status_Grant_ACTIVE},
		},
	}

	actor := &metav1.ObjectReference{Name: "bob", Uid: "uid-bob", ResourceVersion: "3"}

	SetRequestState(req, &StateChange{Status: accessv1.Request_Status_State_PENDING, Reason: accessv1.Request_Status_State_SUBMITTED})
	SetRequestState(req, &StateChange{Status: accessv1.Request_Status_State_PENDING, Reason: accessv1.Request_Status_State_SUBMITTED})
	assert.Empty(t, req.Status.LastStates)

	SetRequestState(req, &StateChange{Status: accessv1.Request_Status_State_PENDING, Reason: accessv1.Request_Status_State_AWAITING_REVIEW})
	require.Len(t, req.Status.LastStates, 1)
	assert.Nil(t, req.Status.ApprovalEndAt)

	SetRequestState(req, &StateChange{
		Status:   accessv1.Request_Status_State_REVOKED,
		Reason:   accessv1.Request_Status_State_ACCESS_REVOKED,
		Message:  "incident closed",
		ActorRef: actor,
	})
	require.Len(t, req.Status.LastStates, 2)
	assert.Equal(t, "incident closed", req.Status.State.Message)
	assert.Empty(t, req.Status.State.ActorRef.ResourceVersion)
	assert.NotNil(t, req.Status.ApprovalEndAt)
	assert.Equal(t, accessv1.Request_Status_Review_StepState_INTERRUPTED, req.Status.Review.Steps[0].Outcome)
	assert.Equal(t, accessv1.Request_Status_Grant_ENDED, req.Status.Grant.State)
	assert.NotNil(t, req.Status.Grant.EndedAt)

	for i := 0; i < MaxRequestStates+10; i++ {
		SetRequestState(req, &StateChange{Status: accessv1.Request_Status_State_Status(1 + i%6), Reason: accessv1.Request_Status_State_Reason(i % 17)})
	}
	assert.Len(t, req.Status.LastStates, MaxRequestStates)
}

func TestIsAccessActive(t *testing.T) {
	now := time.Now()

	newReq := func(grant *accessv1.Request_Status_Grant, startsAt time.Time) *accessv1.Request {
		return &accessv1.Request{
			Status: &accessv1.Request_Status{
				State:          &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_APPROVED},
				AccessStartsAt: pbutils.Timestamp(startsAt),
				AccessEndsAt:   pbutils.Timestamp(now.Add(time.Hour)),
				Grant:          grant,
			},
		}
	}

	assert.True(t, IsAccessActive(newReq(nil, now.Add(-time.Minute)), now))
	assert.True(t, IsAccessActive(newReq(&accessv1.Request_Status_Grant{State: accessv1.Request_Status_Grant_ACTIVE}, now.Add(-time.Minute)), now))
	assert.False(t, IsAccessActive(newReq(&accessv1.Request_Status_Grant{State: accessv1.Request_Status_Grant_SCHEDULED}, now.Add(time.Minute)), now))

	for _, state := range []accessv1.Request_Status_Grant_State{
		accessv1.Request_Status_Grant_PROVISIONING,
		accessv1.Request_Status_Grant_FAILED,
		accessv1.Request_Status_Grant_ENDED,
	} {
		assert.False(t, IsAccessActive(newReq(&accessv1.Request_Status_Grant{State: state}, now.Add(-time.Minute)), now), state.String())
	}
}

func TestIsScopeMateriallyEqual(t *testing.T) {
	nsDefault := &metav1.ObjectReference{Name: "default", Uid: "uid-ns-default"}
	nsProd := &metav1.ObjectReference{Name: "prod", Uid: "uid-ns"}

	newService := func(name, uid string, ns *metav1.ObjectReference, isSystem bool) *corev1.Service {
		return &corev1.Service{
			Metadata: &metav1.Metadata{Name: name, Uid: uid, IsSystem: isSystem},
			Status:   &corev1.Service_Status{NamespaceRef: ns},
		}
	}

	api := newService("api.default", "uid-api", nsDefault, false)
	db := newService("db.prod", "uid-db", nsProd, false)
	cache := newService("cache.prod", "uid-cache", nsProd, false)
	web := newService("web.default", "uid-web", nsDefault, false)
	admin := newService("admin.prod", "uid-admin", nsProd, true)

	newScope := func(services ...*corev1.Service) *Scope {
		ret := &Scope{
			Services: services,
			Status: &accessv1.Request_Status_ResolvedScope{
				CatalogRef:             &metav1.ObjectReference{Name: "prod", Uid: "uid-prod"},
				CatalogResourceVersion: "1",
				CatalogDisplayName:     "Production",
				AccessLevel:            fullLevel("full"),
				Membership:             accessv1.Catalog_Spec_DYNAMIC,
				ResolvedAt:             pbutils.Now(),
				NamespaceRefs:          []*metav1.ObjectReference{nsProd},
			},
		}

		for _, svc := range services {
			ret.Status.ServiceRefs = append(ret.Status.ServiceRefs,
				&metav1.ObjectReference{Name: svc.Metadata.Name, Uid: svc.Metadata.Uid})
		}

		return ret
	}

	a := newScope(api, db)
	b := newScope(api, db, cache)
	b.Status.CatalogResourceVersion = "2"
	b.Status.CatalogDisplayName = "Renamed"
	b.Status.ResolvedAt = pbutils.Timestamp(time.Now().Add(time.Hour))
	b.Status.TotalExcludedService = 3
	assert.True(t, IsScopeMateriallyEqual(a.Status, b))
	assert.False(t, IsScopeEqual(a.Status, b.Status))

	assert.False(t, IsScopeMateriallyEqual(a.Status, newScope(api, db, cache, web)))
	assert.False(t, IsScopeMateriallyEqual(a.Status, newScope(api, db, admin)),
		"a system Service is only granted through its explicit ref")
	assert.True(t, IsScopeMateriallyEqual(newScope(api, db, admin).Status, newScope(api, db, admin)))
	assert.True(t, IsScopeMateriallyEqual(newScope(api, db, admin).Status, newScope(api, db)))
	assert.True(t, IsScopeMateriallyEqual(a.Status, newScope(api)))
	assert.False(t, IsScopeMateriallyEqual(a.Status, newScope(db)))

	c := newScope(api, db)
	c.Status.Priority = -1
	assert.False(t, IsScopeMateriallyEqual(a.Status, c))

	d := newScope(api, db)
	d.Status.AccessLevel.MaxDuration = &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 9}}
	assert.False(t, IsScopeMateriallyEqual(a.Status, d))

	fixed := newScope(api, db)
	fixed.Status.Membership = accessv1.Catalog_Spec_FIXED
	fixed.Status.NamespaceRefs = nil
	fixedMore := newScope(api, db, cache)
	fixedMore.Status.Membership = accessv1.Catalog_Spec_FIXED
	fixedMore.Status.NamespaceRefs = nil
	assert.False(t, IsScopeMateriallyEqual(fixed.Status, fixedMore))

	assert.True(t, IsScopeMateriallyEqual(nil, nil))
	assert.False(t, IsScopeMateriallyEqual(a.Status, nil))
	assert.False(t, IsScopeMateriallyEqual(nil, a))
}

func TestStepRoutingCoversTheRequiredQuorum(t *testing.T) {
	a := newAccessTest(t)

	reviewerA := a.createUser(t)
	reviewerB := a.createUser(t)
	escalation := a.createUser(t)
	fallback := a.createUser(t)
	requester := a.createUser(t)

	userReviewer := func(usr *corev1.User) *Reviewer {
		return &Reviewer{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{
				User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{
					UserRef: umetav1.GetObjectReference(usr),
				},
			},
		}
	}

	newRequest := func(step *accessv1.Policy_Spec_Rule_Action_Review_Step) *accessv1.Request {
		return &accessv1.Request{
			Metadata: &metav1.Metadata{Name: "req", Uid: "uid-req"},
			Spec:     &accessv1.Request_Spec{},
			Status: &accessv1.Request_Status{
				UserRef:      umetav1.GetObjectReference(requester),
				SpecRevision: 1,
				State:        &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_PENDING},
				Rule: &accessv1.Policy_Spec_Rule{
					Action: &accessv1.Policy_Spec_Rule_Action{
						Type: &accessv1.Policy_Spec_Rule_Action_Review_{
							Review: &accessv1.Policy_Spec_Rule_Action_Review{
								Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{step},
							},
						},
					},
				},
			},
		}
	}

	newStep := func() *accessv1.Policy_Spec_Rule_Action_Review_Step {
		return &accessv1.Policy_Spec_Rule_Action_Review_Step{
			ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
			ApprovalCount:       3,
			Reviewers:           []*Reviewer{userReviewer(reviewerA), userReviewer(reviewerB)},
		}
	}

	req := newRequest(newStep())
	require.NoError(t, StartReviewStep(a.ctx, &StartReviewStepOpts{OcteliumC: a.octeliumC, Request: req, Index: 0}))
	stepState := CurrentStepState(req)
	assert.True(t, stepState.IsUnroutable)
	assert.Equal(t, uint32(2), stepState.TotalEligibleReviewer)
	assert.Equal(t, uint32(1), stepState.SpecRevision)
	assert.False(t, IsReviewStepStale(req))

	req.Status.SpecRevision = 2
	assert.True(t, IsReviewStepStale(req))

	withFallback := newRequest(newStep())
	require.NoError(t, StartReviewStep(a.ctx, &StartReviewStepOpts{
		OcteliumC: a.octeliumC,
		Request:   withFallback,
		Index:     0,
		ClusterConfig: &accessv1.ClusterConfig{
			Spec: &accessv1.ClusterConfig_Spec{
				Review: &accessv1.ClusterConfig_Spec_Review{
					FallbackReviewers: []*Reviewer{userReviewer(fallback)},
				},
			},
		},
	}))
	stepState = CurrentStepState(withFallback)
	assert.False(t, stepState.IsUnroutable)
	assert.Equal(t, uint32(3), stepState.TotalEligibleReviewer)

	escalated := newStep()
	escalated.Escalation = &accessv1.Policy_Spec_Rule_Action_Review_Step_Escalation{
		After:     &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
		Reviewers: []*Reviewer{userReviewer(escalation)},
	}
	withEscalation := newRequest(escalated)
	require.NoError(t, StartReviewStep(a.ctx, &StartReviewStepOpts{OcteliumC: a.octeliumC, Request: withEscalation, Index: 0}))
	stepState = CurrentStepState(withEscalation)
	assert.True(t, stepState.IsEscalated)
	assert.False(t, stepState.IsUnroutable)
	assert.Equal(t, uint32(3), stepState.TotalEligibleReviewer)

	stepState.IsEscalated = false
	ok, err := CanReview(a.ctx, a.octeliumC, escalation, withEscalation)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = CanReview(a.ctx, a.octeliumC, fallback, withEscalation)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvalResultBlockingRule(t *testing.T) {
	pol := &accessv1.Policy{Metadata: &metav1.Metadata{Name: "pol"}}
	ruleOf := func(effect accessv1.Policy_Spec_Rule_Effect) *accessv1.Policy_Spec_Rule {
		return &accessv1.Policy_Spec_Rule{Name: effect.String(), Effect: effect}
	}
	failed := func(effect accessv1.Policy_Spec_Rule_Effect) *RuleEvaluation {
		return &RuleEvaluation{
			Policy: pol,
			Rule:   ruleOf(effect),
			Result: accessv1.SimulateRequestResponse_RuleEvaluation_ERROR,
			Err:    fmt.Errorf("broken"),
		}
	}

	for _, tc := range []struct {
		failed  accessv1.Policy_Spec_Rule_Effect
		matched accessv1.Policy_Spec_Rule_Effect
		blocked bool
	}{
		{accessv1.Policy_Spec_Rule_DENY, accessv1.Policy_Spec_Rule_AUTO_APPROVE, true},
		{accessv1.Policy_Spec_Rule_DENY, accessv1.Policy_Spec_Rule_REVIEW, true},
		{accessv1.Policy_Spec_Rule_REVIEW, accessv1.Policy_Spec_Rule_AUTO_APPROVE, true},
		{accessv1.Policy_Spec_Rule_REVIEW, accessv1.Policy_Spec_Rule_REVIEW, true},
		{accessv1.Policy_Spec_Rule_AUTO_APPROVE, accessv1.Policy_Spec_Rule_REVIEW, true},
		{accessv1.Policy_Spec_Rule_AUTO_APPROVE, accessv1.Policy_Spec_Rule_AUTO_APPROVE, true},
		{accessv1.Policy_Spec_Rule_DENY, accessv1.Policy_Spec_Rule_DENY, false},
	} {
		res := &EvalResult{
			Policy: pol,
			Rule:   ruleOf(tc.matched),
			Rules:  []*RuleEvaluation{failed(tc.failed)},
		}
		assert.Equal(t, tc.blocked, res.BlockingRule() != nil, "%s then %s", tc.failed, tc.matched)
	}

	assert.Nil(t, (&EvalResult{Rules: []*RuleEvaluation{failed(accessv1.Policy_Spec_Rule_DENY)}}).BlockingRule())
}

func TestRedactRequest(t *testing.T) {
	req := &accessv1.Request{
		Status: &accessv1.Request_Status{
			Rule:             &accessv1.Policy_Spec_Rule{Name: "secret-rule"},
			PolicyRef:        &metav1.ObjectReference{Name: "pol"},
			PolicyTriggerRef: &metav1.ObjectReference{Name: "pt"},
			ResolvedScope: &accessv1.Request_Status_ResolvedScope{
				AccessLevel: &accessv1.Catalog_Spec_AccessLevel{
					Name: "read",
					Type: &accessv1.Catalog_Spec_AccessLevel_Restricted_{
						Restricted: &accessv1.Catalog_Spec_AccessLevel_Restricted{Policies: []string{"internal"}},
					},
				},
			},
			Evaluation: &accessv1.Request_Status_Evaluation{
				LastError:    "boom",
				SkippedRules: []*accessv1.Request_Status_Evaluation_SkippedRule{{RuleName: "x"}},
			},
		},
	}

	ret := RedactRequest(req)
	assert.Nil(t, ret.Status.Rule)
	assert.Nil(t, ret.Status.PolicyRef)
	assert.Nil(t, ret.Status.PolicyTriggerRef)
	assert.Empty(t, ret.Status.Evaluation.SkippedRules)
	assert.Empty(t, ret.Status.Evaluation.LastError)
	assert.Equal(t, "read", ret.Status.ResolvedScope.AccessLevel.Name)
	require.NotNil(t, ret.Status.ResolvedScope.AccessLevel.GetRestricted())
	assert.Empty(t, ret.Status.ResolvedScope.AccessLevel.GetRestricted().Policies)
	assert.Equal(t, "secret-rule", req.Status.Rule.Name)
	assert.Equal(t, []string{"internal"}, req.Status.ResolvedScope.AccessLevel.GetRestricted().Policies)
}

func TestBuildWorkflow(t *testing.T) {
	a := newAccessTest(t)

	reviewer := a.createUser(t)
	reviewer.Metadata.DisplayName = "Carol"
	reviewer, err := a.octeliumC.CoreC().UpdateUser(a.ctx, reviewer)
	require.NoError(t, err)

	grp, err := a.octeliumC.CoreC().CreateGroup(a.ctx, &corev1.Group{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8), DisplayName: "SRE"},
		Spec:     &corev1.Group_Spec{},
		Status:   &corev1.Group_Status{},
	})
	require.NoError(t, err)

	rule := &accessv1.Policy_Spec_Rule{
		Effect:  accessv1.Policy_Spec_Rule_REVIEW,
		Message: "Two people must agree",
		Action: &accessv1.Policy_Spec_Rule_Action{
			Type: &accessv1.Policy_Spec_Rule_Action_Review_{
				Review: &accessv1.Policy_Spec_Rule_Action_Review{
					Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
						{
							Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
								{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{UserRef: umetav1.GetObjectReference(reviewer)}}},
								{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{GroupRef: umetav1.GetObjectReference(grp)}}},
								{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_{CatalogOwners: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners{}}},
							},
							ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_COUNT,
							ApprovalCount:       2,
							ReasonRequirement:   accessv1.Policy_Spec_Rule_Action_Review_Step_ALWAYS,
							Escalation: &accessv1.Policy_Spec_Rule_Action_Review_Step_Escalation{
								Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
									{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{UserRef: umetav1.GetObjectReference(reviewer)}}},
								},
							},
						},
					},
				},
			},
		},
	}

	workflow, err := BuildWorkflow(a.ctx, a.octeliumC, rule)
	require.NoError(t, err)
	assert.Equal(t, "Two people must agree", workflow.Message)
	require.Len(t, workflow.Steps, 1)

	step := workflow.Steps[0]
	assert.Equal(t, "step-1", step.Name)
	assert.Equal(t, uint32(2), step.ApprovalCount)
	assert.True(t, step.HasEscalation)
	assert.Equal(t, accessv1.Policy_Spec_Rule_Action_Review_Step_ALWAYS, step.ReasonRequirement)
	require.Len(t, step.Reviewers, 3)
	assert.Equal(t, "Carol", step.Reviewers[0].DisplayName)
	assert.Equal(t, "SRE", step.Reviewers[1].DisplayName)
	assert.Equal(t, accessv1.Request_Status_Workflow_Step_Reviewer_CATALOG_OWNERS, step.Reviewers[2].Type)
}

func TestSetReviewDecisionChecks(t *testing.T) {
	a := newAccessTest(t)

	reviewer := a.createUser(t)
	requester := a.createUser(t)
	catalog := a.createCatalog(t, func(spec *accessv1.Catalog_Spec) {
		spec.Owners = &accessv1.Catalog_Spec_Owners{Users: []string{reviewer.Metadata.Name}}
	}, a.createService(t, nil))

	newPending := func(fn func(req *accessv1.Request)) *accessv1.Request {
		req := &accessv1.Request{
			Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
			Spec: &accessv1.Request_Spec{
				Resource: &accessv1.Request_Spec_Resource{
					Type: &accessv1.Request_Spec_Resource_Catalog_{
						Catalog: &accessv1.Request_Spec_Resource_Catalog{CatalogRef: umetav1.GetObjectReference(catalog)},
					},
				},
			},
			Status: &accessv1.Request_Status{
				UserRef:           umetav1.GetObjectReference(requester),
				SpecRevision:      2,
				EffectiveDuration: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 2}},
				State:             &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_PENDING},
				Rule: &accessv1.Policy_Spec_Rule{
					Effect: accessv1.Policy_Spec_Rule_REVIEW,
					Action: &accessv1.Policy_Spec_Rule_Action{
						Type: &accessv1.Policy_Spec_Rule_Action_Review_{
							Review: &accessv1.Policy_Spec_Rule_Action_Review{
								Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
									{
										Name:                "owners",
										ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ANY,
										ReasonRequirement:   accessv1.Policy_Spec_Rule_Action_Review_Step_REJECT_ONLY,
										Timeout:             &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
										Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
											{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners_{CatalogOwners: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_CatalogOwners{}}},
										},
									},
								},
							},
						},
					},
				},
				Review: &accessv1.Request_Status_Review{
					CurrentStepStartedAt: pbutils.Now(),
				},
			},
		}

		if fn != nil {
			fn(req)
		}

		req, err := a.octeliumC.AccessC().CreateRequest(a.ctx, req)
		require.NoError(t, err)
		return req
	}

	decide := func(req *accessv1.Request, opts *SetReviewDecisionOpts) (*accessv1.Review, error) {
		opts.OcteliumC = a.octeliumC
		opts.Request = req
		if opts.Reviewer == nil {
			opts.Reviewer = reviewer
		}
		if opts.ExpectedStepName == "" {
			opts.ExpectedStepName = "owners"
		}
		if opts.ExpectedSpecRevision == 0 {
			opts.ExpectedSpecRevision = req.Status.SpecRevision
		}
		return SetReviewDecision(a.ctx, opts)
	}

	req := newPending(nil)

	_, err := SetReviewDecision(a.ctx, &SetReviewDecisionOpts{
		OcteliumC:            a.octeliumC,
		Request:              req,
		Reviewer:             reviewer,
		Decision:             accessv1.Review_Spec_DECISION_APPROVE,
		ExpectedSpecRevision: req.Status.SpecRevision,
	})
	assert.ErrorContains(t, err, "ExpectedStepName must be set")

	_, err = SetReviewDecision(a.ctx, &SetReviewDecisionOpts{
		OcteliumC:        a.octeliumC,
		Request:          req,
		Reviewer:         reviewer,
		Decision:         accessv1.Review_Spec_DECISION_REJECT,
		Justification:    "not needed",
		ExpectedStepName: "owners",
	})
	assert.ErrorContains(t, err, "ExpectedSpecRevision must be set")

	_, err = decide(req, &SetReviewDecisionOpts{Decision: accessv1.Review_Spec_DECISION_APPROVE, ExpectedStepName: "security"})
	assert.ErrorContains(t, err, "no longer waiting for the security review Step")

	_, err = decide(req, &SetReviewDecisionOpts{Decision: accessv1.Review_Spec_DECISION_REJECT})
	assert.ErrorContains(t, err, "justification is required")

	_, err = decide(req, &SetReviewDecisionOpts{Decision: accessv1.Review_Spec_DECISION_APPROVE, ExpectedSpecRevision: 1})
	assert.ErrorContains(t, err, "was changed after you opened it")

	_, err = decide(req, &SetReviewDecisionOpts{
		Decision: accessv1.Review_Spec_DECISION_APPROVE,
		Duration: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 3}},
	})
	assert.ErrorContains(t, err, "cannot exceed")

	_, err = decide(req, &SetReviewDecisionOpts{
		Decision:      accessv1.Review_Spec_DECISION_REJECT,
		Justification: "not needed",
		Duration:      &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
	})
	assert.ErrorContains(t, err, "only be set for an approval")

	_, err = decide(req, &SetReviewDecisionOpts{Decision: accessv1.Review_Spec_DECISION_APPROVE, Reviewer: requester})
	assert.True(t, grpcerr.IsUnauthorized(err))

	rev, err := decide(req, &SetReviewDecisionOpts{
		Decision:             accessv1.Review_Spec_DECISION_APPROVE,
		ExpectedSpecRevision: 2,
		Duration:             &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
	})
	require.NoError(t, err)
	assert.Equal(t, uint32(2), rev.Status.SpecRevision)
	assert.Equal(t, time.Hour, umetav1.ToDuration(rev.Spec.Duration).ToGo())

	req.Status.SpecRevision = 3
	req, err = a.octeliumC.AccessC().UpdateRequest(a.ctx, req)
	require.NoError(t, err)

	revAgain, err := decide(req, &SetReviewDecisionOpts{
		Decision: accessv1.Review_Spec_DECISION_APPROVE,
		Duration: &metav1.Duration{Type: &metav1.Duration_Hours{Hours: 1}},
	})
	require.NoError(t, err)
	assert.Equal(t, rev.Metadata.Uid, revAgain.Metadata.Uid)
	assert.Equal(t, uint32(3), revAgain.Status.SpecRevision)
	assert.Len(t, revAgain.Status.LastRevisions, 1)

	withdrawn, err := SetReviewDecision(a.ctx, &SetReviewDecisionOpts{
		OcteliumC: a.octeliumC,
		Request:   req,
		Reviewer:  reviewer,
		Decision:  accessv1.Review_Spec_DECISION_UNSET,
	})
	require.NoError(t, err)
	assert.Equal(t, accessv1.Review_Spec_DECISION_UNSET, withdrawn.Spec.Decision)

	timedOut := newPending(func(req *accessv1.Request) {
		req.Status.Review.CurrentStepStartedAt = pbutils.Timestamp(time.Now().Add(-2 * time.Hour))
	})
	_, err = decide(timedOut, &SetReviewDecisionOpts{Decision: accessv1.Review_Spec_DECISION_APPROVE})
	assert.ErrorContains(t, err, "has timed out")

	deadline := newPending(func(req *accessv1.Request) {
		req.Spec.Deadline = pbutils.Timestamp(time.Now().Add(-time.Minute))
	})
	_, err = decide(deadline, &SetReviewDecisionOpts{Decision: accessv1.Review_Spec_DECISION_APPROVE})
	assert.ErrorContains(t, err, "deadline")
}

func TestCanReviewFollowsTheReviewerMembership(t *testing.T) {
	a := newAccessTest(t)

	grp, err := a.octeliumC.CoreC().CreateGroup(a.ctx, &corev1.Group{
		Metadata: &metav1.Metadata{Name: utilrand.GetRandomStringCanonical(8)},
		Spec:     &corev1.Group_Spec{},
		Status:   &corev1.Group_Status{},
	})
	require.NoError(t, err)

	inSnapshot := a.createUser(t, grp.Metadata.Name)
	joinedLater := a.createUser(t, grp.Metadata.Name)
	requester := a.createUser(t, grp.Metadata.Name)

	req := &accessv1.Request{
		Metadata: &metav1.Metadata{Name: "req", Uid: "uid-req"},
		Spec:     &accessv1.Request_Spec{},
		Status: &accessv1.Request_Status{
			UserRef: umetav1.GetObjectReference(requester),
			State:   &accessv1.Request_Status_State{Status: accessv1.Request_Status_State_PENDING},
			Rule: &accessv1.Policy_Spec_Rule{
				Action: &accessv1.Policy_Spec_Rule_Action{
					Type: &accessv1.Policy_Spec_Rule_Action_Review_{
						Review: &accessv1.Policy_Spec_Rule_Action_Review{
							Steps: []*accessv1.Policy_Spec_Rule_Action_Review_Step{
								{
									ApprovalRequirement: accessv1.Policy_Spec_Rule_Action_Review_Step_ALL,
									Reviewers: []*accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer{
										{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group_{Group: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_Group{GroupRef: umetav1.GetObjectReference(grp)}}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, StartReviewStep(a.ctx, &StartReviewStepOpts{
		OcteliumC: a.octeliumC,
		Request:   req,
		Index:     0,
	}))

	stepState := CurrentStepState(req)
	require.NotNil(t, stepState)
	assert.Equal(t, uint32(2), stepState.TotalEligibleReviewer)
	assert.Equal(t, uint32(2), stepState.RequiredApprovals)

	for _, tc := range []struct {
		usr  *corev1.User
		want bool
	}{{inSnapshot, true}, {joinedLater, true}, {requester, false}} {
		ok, err := CanReview(a.ctx, a.octeliumC, tc.usr, req)
		require.NoError(t, err)
		assert.Equal(t, tc.want, ok, tc.usr.Metadata.Name)
	}

	stepState.EligibleReviewers = stepState.EligibleReviewers[:0]
	stepState.EligibleReviewers = append(stepState.EligibleReviewers, umetav1.GetObjectReference(inSnapshot))
	stepState.TotalEligibleReviewer = 1

	ok, err := CanReview(a.ctx, a.octeliumC, joinedLater, req)
	require.NoError(t, err)
	assert.True(t, ok)

	for i := 0; i < MaxStepEligibleReviewers; i++ {
		stepState.EligibleReviewers = append(stepState.EligibleReviewers,
			&metav1.ObjectReference{Uid: utilrand.GetRandomStringCanonical(16)})
	}
	stepState.TotalEligibleReviewer = MaxStepEligibleReviewers + 2

	ok, err = CanReview(a.ctx, a.octeliumC, joinedLater, req)
	require.NoError(t, err)
	assert.True(t, ok)

	inSnapshot.Spec.Groups = nil
	inSnapshot, err = a.octeliumC.CoreC().UpdateUser(a.ctx, inSnapshot)
	require.NoError(t, err)

	ok, err = CanReview(a.ctx, a.octeliumC, inSnapshot, req)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = IsParticipant(a.ctx, a.octeliumC, inSnapshot, req)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = IsParticipant(a.ctx, a.octeliumC, a.createUser(t), req)
	require.NoError(t, err)
	assert.False(t, ok)

	fallback := a.createUser(t)
	cc, err := a.octeliumC.AccessV1Utils().GetClusterConfig(a.ctx)
	require.NoError(t, err)
	if cc.Spec == nil {
		cc.Spec = &accessv1.ClusterConfig_Spec{}
	}
	cc.Spec.Review = &accessv1.ClusterConfig_Spec_Review{
		FallbackReviewers: []*Reviewer{
			{Type: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User_{User: &accessv1.Policy_Spec_Rule_Action_Review_Step_Reviewer_User{UserRef: umetav1.GetObjectReference(fallback)}}},
		},
	}
	_, err = a.octeliumC.AccessC().UpdateClusterConfig(a.ctx, cc)
	require.NoError(t, err)

	ok, err = CanReview(a.ctx, a.octeliumC, fallback, req)
	require.NoError(t, err)
	assert.False(t, ok)

	joinedLater.Spec.IsDisabled = true
	_, err = a.octeliumC.CoreC().UpdateUser(a.ctx, joinedLater)
	require.NoError(t, err)

	ok, err = CanReview(a.ctx, a.octeliumC, fallback, req)
	require.NoError(t, err)
	assert.True(t, ok)

	usrs, err := GetStepReviewerUsers(a.ctx, a.octeliumC,
		req.Status.Rule.GetAction().GetReview().Steps[0], req)
	require.NoError(t, err)
	require.Len(t, usrs, 1)
	assert.Equal(t, fallback.Metadata.Uid, usrs[0].Metadata.Uid)

	EndCurrentStep(req, accessv1.Request_Status_Review_StepState_SATISFIED)
	assert.Nil(t, CurrentStepState(req))
	assert.Equal(t, accessv1.Request_Status_Review_StepState_SATISFIED, GetStepState(req, 0).Outcome)
}

func TestHasReviewBeenApplied(t *testing.T) {
	review := &accessv1.Review{Metadata: &metav1.Metadata{Uid: "rev"}}
	req := &accessv1.Request{
		Status: &accessv1.Request_Status{
			SpecRevision: 2,
			Review: &accessv1.Request_Status_Review{
				CurrentStep: 1,
				LastSteps: []*accessv1.Request_Status_Review_Step{
					{ReviewRef: &metav1.ObjectReference{Uid: "rev"}, StepIndex: 0, SpecRevision: 2},
				},
			},
		},
	}

	assert.True(t, HasReviewBeenApplied(req, review))

	req.Status.SpecRevision = 3
	assert.False(t, HasReviewBeenApplied(req, review))

	req.Status.SpecRevision = 2
	req.Status.Review.CurrentStep = 0
	assert.False(t, HasReviewBeenApplied(req, review))
}
