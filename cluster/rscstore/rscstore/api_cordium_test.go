package rscstore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/ucordiumv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/stretchr/testify/assert"
)

func getTstCordiumNames[T interface{ GetMetadata() *metav1.Metadata }](items []T) []string {
	var ret []string
	for _, itm := range items {
		ret = append(ret, itm.GetMetadata().Name)
	}

	return ret
}

func getTstCordiumRefFilters(ref *metav1.ObjectReference) []*metav1.ObjectReference {
	return []*metav1.ObjectReference{
		{Name: ref.Name},
		{Uid: ref.Uid},
		ref,
	}
}

func TestCordiumListInvalidReferences(t *testing.T) {
	ctx := context.Background()
	srv := &srvCordium{}

	for _, tst := range []struct {
		name    string
		parents int
		list    func(*metav1.ObjectReference) error
	}{
		{
			name:    "workspace space",
			parents: 1,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListWorkspace(ctx, &vcordiumv1.ListWorkspaceOptions{SpaceRef: ref})
				return err
			},
		},
		{
			name:    "workspace template",
			parents: 2,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListWorkspace(ctx, &vcordiumv1.ListWorkspaceOptions{TemplateRef: ref})
				return err
			},
		},
		{
			name:    "template space",
			parents: 1,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListTemplate(ctx, &vcordiumv1.ListTemplateOptions{SpaceRef: ref})
				return err
			},
		},
		{
			name:    "template git provider",
			parents: 2,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListTemplate(ctx, &vcordiumv1.ListTemplateOptions{GitProviderRef: ref})
				return err
			},
		},
		{
			name:    "membership space",
			parents: 1,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListMembership(ctx, &vcordiumv1.ListMembershipOptions{SpaceRef: ref})
				return err
			},
		},
		{
			name:    "git provider space",
			parents: 1,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListGitProvider(ctx, &vcordiumv1.ListGitProviderOptions{SpaceRef: ref})
				return err
			},
		},
		{
			name:    "secret space",
			parents: 1,
			list: func(ref *metav1.ObjectReference) error {
				_, err := srv.ListSecret(ctx, &vcordiumv1.ListSecretOptions{SpaceRef: ref})
				return err
			},
		},
	} {
		for _, ref := range []*metav1.ObjectReference{
			{},
			{Name: "invalid name"},
			{Name: ".space"},
			{Name: "space."},
			{Name: "space..user"},
			{Name: strings.Repeat("parent.", tst.parents+1) + "resource"},
			{Uid: "not-a-uid"},
		} {
			assert.NotNil(t, tst.list(ref), "%s: %+v", tst.name, ref)
		}
	}
}

func TestCordiumListWorkspaceFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvCordium{s: env.srv}
	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertTstCordiumWorkspaces(t, env, refs, now)

	list := func(req *vcordiumv1.ListWorkspaceOptions) []string {
		t.Helper()

		req.Common = &vmetav1.CommonListOptions{
			OrderBy: &vmetav1.CommonListOptions_OrderBy{
				Type: vmetav1.CommonListOptions_OrderBy_NAME,
				Mode: vmetav1.CommonListOptions_OrderBy_ASC,
			},
		}

		resp, err := srv.ListWorkspace(env.ctx, req)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return nil
		}

		assert.Equal(t, ucordiumv1.APIVersion, resp.ApiVersion)
		assert.Equal(t, "WorkspaceList", resp.Kind)

		return getTstCordiumNames(resp.Items)
	}

	assert.Equal(t, []string{"ws-failed", "ws-init", "ws-pulling", "ws-running", "ws-stopping"},
		list(&vcordiumv1.ListWorkspaceOptions{}))

	assert.Equal(t, []string{"ws-failed", "ws-running"},
		list(&vcordiumv1.ListWorkspaceOptions{UserRef: refs.userOne}))
	assert.Equal(t, []string{"ws-init"},
		list(&vcordiumv1.ListWorkspaceOptions{UserRef: &metav1.ObjectReference{Name: refs.userTwo.Name}}))
	assert.Equal(t, []string{"ws-running"},
		list(&vcordiumv1.ListWorkspaceOptions{SessionRef: refs.session}))
	for _, ref := range getTstCordiumRefFilters(refs.spaceOne) {
		assert.Equal(t, []string{"ws-init", "ws-running"},
			list(&vcordiumv1.ListWorkspaceOptions{SpaceRef: ref}))
	}
	for _, ref := range getTstCordiumRefFilters(refs.spaceTwo) {
		assert.Equal(t, []string{"ws-failed"},
			list(&vcordiumv1.ListWorkspaceOptions{SpaceRef: ref}))
	}
	for _, ref := range getTstCordiumRefFilters(refs.template) {
		assert.Equal(t, []string{"ws-running"},
			list(&vcordiumv1.ListWorkspaceOptions{TemplateRef: ref}))
	}
	assert.Equal(t, []string{"ws-init"},
		list(&vcordiumv1.ListWorkspaceOptions{RegionRef: refs.regionTwo}))

	assert.Equal(t, []string{"ws-running"},
		list(&vcordiumv1.ListWorkspaceOptions{State: cordiumv1.Workspace_Status_RUNNING}))
	assert.Equal(t, []string{"ws-stopping"},
		list(&vcordiumv1.ListWorkspaceOptions{State: cordiumv1.Workspace_Status_STOPPING_REQUEST}))
	assert.Equal(t, []string{"ws-failed"},
		list(&vcordiumv1.ListWorkspaceOptions{SpaceType: cordiumv1.Space_Status_ORGANIZATION}))
	assert.Equal(t, []string{"ws-init", "ws-running"},
		list(&vcordiumv1.ListWorkspaceOptions{SpaceType: cordiumv1.Space_Status_USER}))
	assert.Equal(t, []string{"ws-failed"},
		list(&vcordiumv1.ListWorkspaceOptions{StoppingReason: cordiumv1.Workspace_Status_STOPPING_REASON_ERROR}))
	assert.Equal(t, []string{"ws-stopping"},
		list(&vcordiumv1.ListWorkspaceOptions{StoppingReason: cordiumv1.Workspace_Status_STOPPING_REASON_API}))

	assert.Equal(t, []string{"ws-failed"},
		list(&vcordiumv1.ListWorkspaceOptions{IsBuild: true}))
	assert.Equal(t, []string{"ws-running"},
		list(&vcordiumv1.ListWorkspaceOptions{IsEphemeral: true}))
	assert.Equal(t, []string{"ws-failed", "ws-running"},
		list(&vcordiumv1.ListWorkspaceOptions{IsShared: true}))
	assert.Equal(t, []string{"ws-failed"},
		list(&vcordiumv1.ListWorkspaceOptions{IsFailed: true}))

	assert.Nil(t, list(&vcordiumv1.ListWorkspaceOptions{
		UserRef:  refs.userTwo,
		IsFailed: true,
	}))

	{
		_, err := srv.ListWorkspace(env.ctx, &vcordiumv1.ListWorkspaceOptions{
			UserRef: &metav1.ObjectReference{Uid: "not-a-uid"},
		})
		assert.NotNil(t, err)
	}
}

func TestCordiumListTemplateFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvCordium{s: env.srv}
	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertTstCordiumTemplates(t, env, refs, now)

	list := func(req *vcordiumv1.ListTemplateOptions) []string {
		t.Helper()

		req.Common = &vmetav1.CommonListOptions{
			OrderBy: &vmetav1.CommonListOptions_OrderBy{
				Type: vmetav1.CommonListOptions_OrderBy_NAME,
				Mode: vmetav1.CommonListOptions_OrderBy_ASC,
			},
		}

		resp, err := srv.ListTemplate(env.ctx, req)
		assert.Nil(t, err, "%+v", err)
		if err != nil {
			return nil
		}

		return getTstCordiumNames(resp.Items)
	}

	assert.Equal(t, []string{"tmpl-building", "tmpl-new", "tmpl-ready"},
		list(&vcordiumv1.ListTemplateOptions{}))
	assert.Equal(t, []string{"tmpl-ready"},
		list(&vcordiumv1.ListTemplateOptions{UserRef: refs.userOne}))
	for _, ref := range getTstCordiumRefFilters(refs.spaceOne) {
		assert.Equal(t, []string{"tmpl-building", "tmpl-ready"},
			list(&vcordiumv1.ListTemplateOptions{SpaceRef: ref}))
	}
	for _, ref := range getTstCordiumRefFilters(refs.gitProvider) {
		assert.Equal(t, []string{"tmpl-ready"},
			list(&vcordiumv1.ListTemplateOptions{GitProviderRef: ref}))
	}

	assert.Equal(t, []string{"tmpl-ready"},
		list(&vcordiumv1.ListTemplateOptions{
			BuildState: cordiumv1.Template_Status_BuildInfo_Build_STATE_READY,
		}))
	assert.Equal(t, []string{"tmpl-building"},
		list(&vcordiumv1.ListTemplateOptions{
			BuildState: cordiumv1.Template_Status_BuildInfo_Build_STATE_RUNNING,
		}))
	assert.Nil(t, list(&vcordiumv1.ListTemplateOptions{
		BuildState: cordiumv1.Template_Status_BuildInfo_Build_STATE_FAILED,
	}))

	assert.Equal(t, []string{"tmpl-ready"},
		list(&vcordiumv1.ListTemplateOptions{HasReadyBuild: true}))
	assert.Equal(t, []string{"tmpl-building"},
		list(&vcordiumv1.ListTemplateOptions{IsBuilding: true}))
}

func TestCordiumListSpaceAndMembershipFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvCordium{s: env.srv}
	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertRscStoreObject(t, env, newTstSpace(refs.spaceOne, now,
		&cordiumv1.Space_Spec{
			Authorization: &cordiumv1.Space_Spec_Authorization{
				DisableSSH: true,
			},
		},
		&cordiumv1.Space_Status{
			Type:    cordiumv1.Space_Status_USER,
			UserRef: refs.userOne,
		}))
	insertRscStoreObject(t, env, newTstSpace(refs.spaceTwo, now.Add(time.Second), nil,
		&cordiumv1.Space_Status{
			Type:    cordiumv1.Space_Status_ORGANIZATION,
			UserRef: refs.userTwo,
		}))

	{
		resp, err := srv.ListSpace(env.ctx, &vcordiumv1.ListSpaceOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 2)
	}

	{
		resp, err := srv.ListSpace(env.ctx, &vcordiumv1.ListSpaceOptions{UserRef: refs.userTwo})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{refs.spaceTwo.Name}, getTstCordiumNames(resp.Items))
	}

	{
		resp, err := srv.ListSpace(env.ctx, &vcordiumv1.ListSpaceOptions{Type: cordiumv1.Space_Status_USER})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{refs.spaceOne.Name}, getTstCordiumNames(resp.Items))
	}

	{
		resp, err := srv.ListSpace(env.ctx, &vcordiumv1.ListSpaceOptions{IsSSHDisabled: true})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{refs.spaceOne.Name}, getTstCordiumNames(resp.Items))
	}

	insertRscStoreObject(t, env, newTstMembership("mem-owner", now,
		cordiumv1.Membership_Spec_OWNER, refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstMembership("mem-admin", now.Add(time.Second),
		cordiumv1.Membership_Spec_ADMIN, refs.userTwo, refs.spaceOne))
	insertRscStoreObject(t, env, newTstMembership("mem-user", now.Add(2*time.Second),
		cordiumv1.Membership_Spec_USER, refs.userTwo, refs.spaceTwo))

	{
		resp, err := srv.ListMembership(env.ctx, &vcordiumv1.ListMembershipOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 3)
		assert.EqualValues(t, 3, resp.ListResponseMeta.TotalCount)
	}

	for _, ref := range getTstCordiumRefFilters(refs.spaceOne) {
		resp, err := srv.ListMembership(env.ctx, &vcordiumv1.ListMembershipOptions{SpaceRef: ref})
		assert.Nil(t, err, "%+v", err)
		if err == nil {
			assert.ElementsMatch(t, []string{"mem-owner", "mem-admin"}, getTstCordiumNames(resp.Items))
		}
	}

	{
		resp, err := srv.ListMembership(env.ctx, &vcordiumv1.ListMembershipOptions{UserRef: refs.userTwo})
		assert.Nil(t, err, "%+v", err)
		assert.ElementsMatch(t, []string{"mem-admin", "mem-user"}, getTstCordiumNames(resp.Items))
	}

	{
		resp, err := srv.ListMembership(env.ctx, &vcordiumv1.ListMembershipOptions{
			Role: cordiumv1.Membership_Spec_ADMIN,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"mem-admin"}, getTstCordiumNames(resp.Items))
	}
}

func TestCordiumListGitProviderSecretAndUserSecretFilters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvCordium{s: env.srv}
	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertRscStoreObject(t, env, newTstGitProvider("github-one", now,
		&cordiumv1.GitProvider_Spec{
			Type: &cordiumv1.GitProvider_Spec_Github_{
				Github: &cordiumv1.GitProvider_Spec_Github{ClientID: "client-one"},
			},
		}, refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstGitProvider("gitlab-one", now.Add(time.Second),
		&cordiumv1.GitProvider_Spec{
			Type: &cordiumv1.GitProvider_Spec_Gitlab_{
				Gitlab: &cordiumv1.GitProvider_Spec_Gitlab{ClientID: "client-two"},
			},
		}, refs.userTwo, refs.spaceTwo))
	insertRscStoreObject(t, env, newTstGitProvider("oauth2-one", now.Add(2*time.Second),
		&cordiumv1.GitProvider_Spec{
			Type: &cordiumv1.GitProvider_Spec_Oauth2{
				Oauth2: &cordiumv1.GitProvider_Spec_OAuth2{ClientID: "client-three"},
			},
		}, nil, refs.spaceOne))

	for typ, name := range map[vcordiumv1.ListGitProviderOptions_Type]string{
		vcordiumv1.ListGitProviderOptions_GITHUB: "github-one",
		vcordiumv1.ListGitProviderOptions_GITLAB: "gitlab-one",
		vcordiumv1.ListGitProviderOptions_OAUTH2: "oauth2-one",
	} {
		resp, err := srv.ListGitProvider(env.ctx, &vcordiumv1.ListGitProviderOptions{Type: typ})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{name}, getTstCordiumNames(resp.Items), typ.String())
	}

	for _, ref := range getTstCordiumRefFilters(refs.spaceOne) {
		resp, err := srv.ListGitProvider(env.ctx, &vcordiumv1.ListGitProviderOptions{SpaceRef: ref})
		assert.Nil(t, err, "%+v", err)
		if err == nil {
			assert.ElementsMatch(t, []string{"github-one", "oauth2-one"}, getTstCordiumNames(resp.Items))
		}
	}

	{
		resp, err := srv.ListGitProvider(env.ctx, &vcordiumv1.ListGitProviderOptions{UserRef: refs.userTwo})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"gitlab-one"}, getTstCordiumNames(resp.Items))
	}

	insertRscStoreObject(t, env, newTstCordiumSecret("secret-one", now, "v1", refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstCordiumSecret("secret-two", now.Add(time.Second), "v2", refs.userTwo, refs.spaceTwo))

	for _, ref := range getTstCordiumRefFilters(refs.spaceTwo) {
		resp, err := srv.ListSecret(env.ctx, &vcordiumv1.ListSecretOptions{SpaceRef: ref})
		assert.Nil(t, err, "%+v", err)
		if err == nil {
			assert.Equal(t, []string{"secret-two"}, getTstCordiumNames(resp.Items))
		}
	}

	{
		resp, err := srv.ListSecret(env.ctx, &vcordiumv1.ListSecretOptions{UserRef: refs.userOne})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"secret-one"}, getTstCordiumNames(resp.Items))
	}

	insertRscStoreObject(t, env, newTstUserSecret("usec-default", now,
		cordiumv1.UserSecret_Spec_DEFAULT, "v1", refs.userOne))
	insertRscStoreObject(t, env, newTstUserSecret("usec-ssh", now.Add(time.Second),
		cordiumv1.UserSecret_Spec_SSH_KEY, "k1", refs.userTwo))

	{
		resp, err := srv.ListUserSecret(env.ctx, &vcordiumv1.ListUserSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 2)
	}

	{
		resp, err := srv.ListUserSecret(env.ctx, &vcordiumv1.ListUserSecretOptions{
			Type: cordiumv1.UserSecret_Spec_SSH_KEY,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"usec-ssh"}, getTstCordiumNames(resp.Items))
		assert.Equal(t, "ssh-ed25519 AAAA", resp.Items[0].Status.GetSshKey().GetPublicKey())
	}

	{
		resp, err := srv.ListUserSecret(env.ctx, &vcordiumv1.ListUserSecretOptions{UserRef: refs.userOne})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"usec-default"}, getTstCordiumNames(resp.Items))
	}
}

func TestCordiumListRegion(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvCordium{s: env.srv}
	now := time.Now().UTC()

	enabled := newTstRegion("rgn-enabled", now, true)
	enabled.Metadata.DisplayName = "Enabled Region"
	insertRscStoreObject(t, env, enabled)
	insertRscStoreObject(t, env, newTstRegion("rgn-disabled", now.Add(time.Second), false))

	{
		resp, err := srv.ListRegion(env.ctx, &vcordiumv1.ListRegionOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, ucordiumv1.APIVersion, resp.ApiVersion)
		assert.Equal(t, "RegionList", resp.Kind)
		assert.ElementsMatch(t, []string{"rgn-enabled", "rgn-disabled"}, getTstCordiumNames(resp.Items))
		assert.EqualValues(t, 2, resp.ListResponseMeta.TotalCount)

		for _, itm := range resp.Items {
			assert.Equal(t, ucordiumv1.APIVersion, itm.ApiVersion)
			assert.Equal(t, "Region", itm.Kind)
			assert.NotNil(t, itm.Spec)
			assert.NotNil(t, itm.Status)
		}
	}

	{
		resp, err := srv.ListRegion(env.ctx, &vcordiumv1.ListRegionOptions{IsEnabled: true})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "rgn-enabled", resp.Items[0].Metadata.Name)
		assert.Equal(t, enabled.Metadata.Uid, resp.Items[0].Metadata.Uid)
		assert.Equal(t, "Enabled Region", resp.Items[0].Metadata.DisplayName)
	}

	{
		resp, err := srv.ListRegion(env.ctx, &vcordiumv1.ListRegionOptions{
			Common: &vmetav1.CommonListOptions{
				Query: "rgn-disabled",
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"rgn-disabled"}, getTstCordiumNames(resp.Items))
	}
}
