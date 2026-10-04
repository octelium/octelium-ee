package rscstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/ucordiumv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

type tstCordiumRefs struct {
	userOne     *metav1.ObjectReference
	userTwo     *metav1.ObjectReference
	session     *metav1.ObjectReference
	spaceOne    *metav1.ObjectReference
	spaceTwo    *metav1.ObjectReference
	template    *metav1.ObjectReference
	regionOne   *metav1.ObjectReference
	regionTwo   *metav1.ObjectReference
	gitProvider *metav1.ObjectReference
}

func newTstCordiumRefs() *tstCordiumRefs {
	return &tstCordiumRefs{
		userOne:     &metav1.ObjectReference{Name: "user-one", Uid: vutils.UUIDv4()},
		userTwo:     &metav1.ObjectReference{Name: "user-two", Uid: vutils.UUIDv4()},
		session:     &metav1.ObjectReference{Name: "session-one", Uid: vutils.UUIDv4()},
		spaceOne:    &metav1.ObjectReference{Name: "space-one.user-one", Uid: vutils.UUIDv4()},
		spaceTwo:    &metav1.ObjectReference{Name: "space-two.cordium", Uid: vutils.UUIDv4()},
		template:    &metav1.ObjectReference{Name: "template-one.space-one.user-one", Uid: vutils.UUIDv4()},
		regionOne:   &metav1.ObjectReference{Name: "region-one", Uid: vutils.UUIDv4()},
		regionTwo:   &metav1.ObjectReference{Name: "region-two", Uid: vutils.UUIDv4()},
		gitProvider: &metav1.ObjectReference{Name: "github-one.space-one.user-one", Uid: vutils.UUIDv4()},
	}
}

func newTstWorkspace(name string, createdAt time.Time,
	spec *cordiumv1.Workspace_Spec, status *cordiumv1.Workspace_Status) *cordiumv1.Workspace {
	if spec == nil {
		spec = &cordiumv1.Workspace_Spec{}
	}

	return &cordiumv1.Workspace{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindWorkspace,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       spec,
		Status:     status,
	}
}

func newTstWorkspaceLimit(cpu, memory, storage uint32) *cordiumv1.Workspace_Spec_Limit {
	return &cordiumv1.Workspace_Spec_Limit{
		Cpu:     &cordiumv1.Workspace_Spec_Limit_CPU{Millicores: cpu},
		Memory:  &cordiumv1.Workspace_Spec_Limit_Memory{Megabytes: memory},
		Storage: &cordiumv1.Workspace_Spec_Limit_Storage{Megabytes: storage},
	}
}

func newTstTemplate(name string, createdAt time.Time, status *cordiumv1.Template_Status) *cordiumv1.Template {
	return &cordiumv1.Template{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindTemplate,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       &cordiumv1.Template_Spec{},
		Status:     status,
	}
}

func newTstSpace(ref *metav1.ObjectReference, createdAt time.Time,
	spec *cordiumv1.Space_Spec, status *cordiumv1.Space_Status) *cordiumv1.Space {
	if spec == nil {
		spec = &cordiumv1.Space_Spec{}
	}

	md := newRscStoreMetadata(ref.Name, createdAt)
	md.Uid = ref.Uid

	return &cordiumv1.Space{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindSpace,
		Metadata:   md,
		Spec:       spec,
		Status:     status,
	}
}

func newTstMembership(name string, createdAt time.Time, role cordiumv1.Membership_Spec_Role,
	userRef, spaceRef *metav1.ObjectReference) *cordiumv1.Membership {
	return &cordiumv1.Membership{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindMembership,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec: &cordiumv1.Membership_Spec{
			Role: role,
		},
		Status: &cordiumv1.Membership_Status{
			UserRef:  userRef,
			SpaceRef: spaceRef,
		},
	}
}

func newTstGitProvider(name string, createdAt time.Time, spec *cordiumv1.GitProvider_Spec,
	userRef, spaceRef *metav1.ObjectReference) *cordiumv1.GitProvider {
	return &cordiumv1.GitProvider{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindGitProvider,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       spec,
		Status: &cordiumv1.GitProvider_Status{
			UserRef:  userRef,
			SpaceRef: spaceRef,
		},
	}
}

func newTstCordiumSecret(name string, createdAt time.Time, value string,
	userRef, spaceRef *metav1.ObjectReference) *cordiumv1.Secret {
	return &cordiumv1.Secret{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindSecret,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       &cordiumv1.Secret_Spec{},
		Status: &cordiumv1.Secret_Status{
			UserRef:  userRef,
			SpaceRef: spaceRef,
		},
		Data: &cordiumv1.Secret_Data{
			Type: &cordiumv1.Secret_Data_Value{
				Value: value,
			},
		},
	}
}

func newTstUserSecret(name string, createdAt time.Time, typ cordiumv1.UserSecret_Spec_Type,
	value string, userRef *metav1.ObjectReference) *cordiumv1.UserSecret {
	ret := &cordiumv1.UserSecret{
		ApiVersion: ucordiumv1.APIVersion,
		Kind:       ucordiumv1.KindUserSecret,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec: &cordiumv1.UserSecret_Spec{
			Type: typ,
		},
		Status: &cordiumv1.UserSecret_Status{
			UserRef: userRef,
		},
		Data: &cordiumv1.UserSecret_Data{
			Type: &cordiumv1.UserSecret_Data_Value{
				Value: value,
			},
		},
	}

	if typ == cordiumv1.UserSecret_Spec_SSH_KEY {
		ret.Status.Details = &cordiumv1.UserSecret_Status_SshKey{
			SshKey: &cordiumv1.UserSecret_Status_SSHKey{
				PublicKey: "ssh-ed25519 AAAA",
			},
		}
	}

	return ret
}

func newTstRegion(name string, createdAt time.Time, isEnabled bool) *corev1.Region {
	ret := &corev1.Region{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindRegion,
		Metadata:   newRscStoreMetadata(name, createdAt),
		Spec:       &corev1.Region_Spec{},
		Status:     &corev1.Region_Status{},
	}

	if isEnabled {
		ret.Status.Ext = map[string]*structpb.Struct{
			"cordium": pbutils.MessageToStructMust(&cordiumv1.RegionExtInfo{
				IsEnabled: true,
			}),
		}
	}

	return ret
}

func insertTstCordiumWorkspaces(t *testing.T, env *rscStoreTestEnv, refs *tstCordiumRefs, now time.Time) {
	t.Helper()

	insertRscStoreObject(t, env, newTstWorkspace("ws-running", now,
		&cordiumv1.Workspace_Spec{
			IsEphemeral: true,
			Runtime: &cordiumv1.Workspace_Spec_Runtime{
				AutoStop: true,
			},
		},
		&cordiumv1.Workspace_Status{
			State:       cordiumv1.Workspace_Status_RUNNING,
			UserRef:     refs.userOne,
			SessionRef:  refs.session,
			RegionRef:   refs.regionOne,
			TemplateRef: refs.template,
			SpaceRef:    refs.spaceOne,
			SpaceType:   cordiumv1.Space_Status_USER,
			Limit:       newTstWorkspaceLimit(2000, 4096, 10240),
			SharedPorts: []*cordiumv1.Workspace_Status_SharedPort{
				{Mode: cordiumv1.Workspace_Status_SharedPort_MEMBERS, ApplicationName: "web"},
			},
			Run: &cordiumv1.Workspace_Status_Run{
				Id: "run-one",
			},
		}))

	insertRscStoreObject(t, env, newTstWorkspace("ws-failed", now.Add(time.Second), nil,
		&cordiumv1.Workspace_Status{
			State:          cordiumv1.Workspace_Status_STOPPED,
			StoppingReason: cordiumv1.Workspace_Status_STOPPING_REASON_ERROR,
			UserRef:        refs.userOne,
			SpaceRef:       refs.spaceTwo,
			SpaceType:      cordiumv1.Space_Status_ORGANIZATION,
			IsBuild:        true,
			Limit:          newTstWorkspaceLimit(1000, 1024, 1024),
			SharedPorts: []*cordiumv1.Workspace_Status_SharedPort{
				{Mode: cordiumv1.Workspace_Status_SharedPort_ALL, ApplicationName: "api"},
			},
			Run: &cordiumv1.Workspace_Status_Run{
				Id: "run-two",
				Failure: &cordiumv1.Workspace_Status_Failure{
					Message: "could not pull the image",
					Type: &cordiumv1.Workspace_Status_Failure_ImagePull_{
						ImagePull: &cordiumv1.Workspace_Status_Failure_ImagePull{},
					},
				},
			},
		}))

	insertRscStoreObject(t, env, newTstWorkspace("ws-init", now.Add(2*time.Second), nil,
		&cordiumv1.Workspace_Status{
			State:     cordiumv1.Workspace_Status_INITIALIZING,
			UserRef:   refs.userTwo,
			SpaceRef:  refs.spaceOne,
			SpaceType: cordiumv1.Space_Status_USER,
			RegionRef: refs.regionTwo,
			Limit:     newTstWorkspaceLimit(500, 1024, 2048),
		}))

	insertRscStoreObject(t, env, newTstWorkspace("ws-stopping", now.Add(3*time.Second), nil,
		&cordiumv1.Workspace_Status{
			State:          cordiumv1.Workspace_Status_STOPPING_REQUEST,
			StoppingReason: cordiumv1.Workspace_Status_STOPPING_REASON_API,
			Limit: &cordiumv1.Workspace_Spec_Limit{
				Cpu: &cordiumv1.Workspace_Spec_Limit_CPU{Millicores: 250},
			},
		}))

	insertRscStoreObject(t, env, newTstWorkspace("ws-pulling", now.Add(4*time.Second), nil,
		&cordiumv1.Workspace_Status{
			State: cordiumv1.Workspace_Status_PULLING_IMAGE,
		}))
}

func insertTstCordiumTemplates(t *testing.T, env *rscStoreTestEnv, refs *tstCordiumRefs, now time.Time) {
	t.Helper()

	insertRscStoreObject(t, env, newTstTemplate("tmpl-ready", now,
		&cordiumv1.Template_Status{
			UserRef:        refs.userOne,
			SpaceRef:       refs.spaceOne,
			GitProviderRef: refs.gitProvider,
			BuildInfo: &cordiumv1.Template_Status_BuildInfo{
				Builds: []*cordiumv1.Template_Status_BuildInfo_Build{
					{Id: "b2", State: cordiumv1.Template_Status_BuildInfo_Build_STATE_READY},
					{Id: "b1", State: cordiumv1.Template_Status_BuildInfo_Build_STATE_FAILED},
				},
				CurrentReadyBuildID: "b2",
			},
		}))

	insertRscStoreObject(t, env, newTstTemplate("tmpl-building", now.Add(time.Second),
		&cordiumv1.Template_Status{
			UserRef:  refs.userTwo,
			SpaceRef: refs.spaceOne,
			BuildInfo: &cordiumv1.Template_Status_BuildInfo{
				Builds: []*cordiumv1.Template_Status_BuildInfo_Build{
					{Id: "b4", State: cordiumv1.Template_Status_BuildInfo_Build_STATE_RUNNING},
					{Id: "b3", State: cordiumv1.Template_Status_BuildInfo_Build_STATE_FAILED, IsCanceled: true},
				},
				CurrentRunningBuildID: "b4",
			},
		}))

	insertRscStoreObject(t, env, newTstTemplate("tmpl-new", now.Add(2*time.Second),
		&cordiumv1.Template_Status{
			SpaceRef: refs.spaceTwo,
		}))
}

func TestCordiumSummaries(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertTstCordiumWorkspaces(t, env, refs, now)

	{
		resp, err := env.srv.getSummaryCordiumWorkspace(env.ctx, &vcordiumv1.GetWorkspaceSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 5, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalInitializing)
		assert.EqualValues(t, 1, resp.TotalPullingImage)
		assert.EqualValues(t, 0, resp.TotalBuildingImage)
		assert.EqualValues(t, 0, resp.TotalStartingRuntime)
		assert.EqualValues(t, 0, resp.TotalPreparing)
		assert.EqualValues(t, 1, resp.TotalRunning)
		assert.EqualValues(t, 1, resp.TotalStopping)
		assert.EqualValues(t, 1, resp.TotalStopped)
		assert.EqualValues(t, 1, resp.TotalFailed)
		assert.EqualValues(t, 1, resp.TotalBuild)
		assert.EqualValues(t, 1, resp.TotalEphemeral)
		assert.EqualValues(t, 1, resp.TotalAutoStop)
		assert.EqualValues(t, 2, resp.TotalShared)
		assert.EqualValues(t, 1, resp.TotalSharedMembers)
		assert.EqualValues(t, 1, resp.TotalSharedAll)
		assert.EqualValues(t, 2, resp.TotalUserSpace)
		assert.EqualValues(t, 1, resp.TotalOrganizationSpace)
		assert.EqualValues(t, 2, resp.TotalUser)
		assert.EqualValues(t, 1, resp.TotalSession)
		assert.EqualValues(t, 2, resp.TotalSpace)
		assert.EqualValues(t, 1, resp.TotalTemplate)
		assert.EqualValues(t, 2, resp.TotalRegion)
		assert.EqualValues(t, 2750, resp.TotalCPUMillicores)
		assert.EqualValues(t, 5120, resp.TotalMemoryMegabytes)
		assert.EqualValues(t, 12288, resp.TotalStorageMegabytes)
		assert.Nil(t, resp.Previous)
	}

	insertTstCordiumTemplates(t, env, refs, now)

	{
		resp, err := env.srv.getSummaryCordiumTemplate(env.ctx, &vcordiumv1.GetTemplateSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
		assert.EqualValues(t, 4, resp.TotalBuild)
		assert.EqualValues(t, 1, resp.TotalBuildReady)
		assert.EqualValues(t, 1, resp.TotalBuildRunning)
		assert.EqualValues(t, 2, resp.TotalBuildFailed)
		assert.EqualValues(t, 1, resp.TotalBuildCanceled)
		assert.EqualValues(t, 1, resp.TotalWithReadyBuild)
		assert.EqualValues(t, 1, resp.TotalWithRunningBuild)
		assert.EqualValues(t, 1, resp.TotalNeverBuilt)
		assert.EqualValues(t, 2, resp.TotalUser)
		assert.EqualValues(t, 2, resp.TotalSpace)
		assert.EqualValues(t, 1, resp.TotalGitProvider)
	}

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
	insertRscStoreObject(t, env, newTstSpace(refs.spaceTwo, now.Add(-2*time.Hour), nil,
		&cordiumv1.Space_Status{
			Type:    cordiumv1.Space_Status_ORGANIZATION,
			UserRef: refs.userTwo,
		}))

	insertRscStoreObject(t, env, newTstMembership("mem-owner", now,
		cordiumv1.Membership_Spec_OWNER, refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstMembership("mem-admin", now,
		cordiumv1.Membership_Spec_ADMIN, refs.userTwo, refs.spaceOne))
	insertRscStoreObject(t, env, newTstMembership("mem-user", now,
		cordiumv1.Membership_Spec_USER, refs.userTwo, refs.spaceTwo))

	{
		resp, err := env.srv.getSummaryCordiumSpace(env.ctx, &vcordiumv1.GetSpaceSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalUserSpace)
		assert.EqualValues(t, 1, resp.TotalOrganizationSpace)
		assert.EqualValues(t, 1, resp.TotalSSHDisabled)
		assert.EqualValues(t, 2, resp.TotalUser)
		assert.EqualValues(t, 3, resp.TotalMembership)
		assert.EqualValues(t, 3, resp.TotalWorkspace)
		assert.EqualValues(t, 3, resp.TotalTemplate)
	}

	{
		resp, err := env.srv.getSummaryCordiumSpace(env.ctx, &vcordiumv1.GetSpaceSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From: pbutils.Timestamp(now.Add(-time.Hour)),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 1, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalUserSpace)
		assert.EqualValues(t, 0, resp.TotalOrganizationSpace)
		assert.EqualValues(t, 2, resp.TotalMembership)
		assert.EqualValues(t, 2, resp.TotalWorkspace)
		assert.EqualValues(t, 2, resp.TotalTemplate)
	}

	{
		resp, err := env.srv.getSummaryCordiumMembership(env.ctx, &vcordiumv1.GetMembershipSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalRoleOwner)
		assert.EqualValues(t, 1, resp.TotalRoleAdmin)
		assert.EqualValues(t, 1, resp.TotalRoleUser)
		assert.EqualValues(t, 2, resp.TotalUser)
		assert.EqualValues(t, 2, resp.TotalSpace)
	}

	insertRscStoreObject(t, env, newTstGitProvider("github-one", now,
		&cordiumv1.GitProvider_Spec{
			Type: &cordiumv1.GitProvider_Spec_Github_{
				Github: &cordiumv1.GitProvider_Spec_Github{ClientID: "client-one"},
			},
		}, refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstGitProvider("gitlab-one", now,
		&cordiumv1.GitProvider_Spec{
			Type: &cordiumv1.GitProvider_Spec_Gitlab_{
				Gitlab: &cordiumv1.GitProvider_Spec_Gitlab{ClientID: "client-two"},
			},
		}, refs.userOne, refs.spaceTwo))
	insertRscStoreObject(t, env, newTstGitProvider("oauth2-one", now,
		&cordiumv1.GitProvider_Spec{
			Type: &cordiumv1.GitProvider_Spec_Oauth2{
				Oauth2: &cordiumv1.GitProvider_Spec_OAuth2{ClientID: "client-three"},
			},
		}, nil, refs.spaceOne))

	{
		resp, err := env.srv.getSummaryCordiumGitProvider(env.ctx, &vcordiumv1.GetGitProviderSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalGithub)
		assert.EqualValues(t, 1, resp.TotalGitlab)
		assert.EqualValues(t, 1, resp.TotalOAuth2)
		assert.EqualValues(t, 1, resp.TotalUser)
		assert.EqualValues(t, 2, resp.TotalSpace)
	}

	insertRscStoreObject(t, env, newTstCordiumSecret("secret-one", now, "s3cr3t-one", refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstCordiumSecret("secret-two", now, "s3cr3t-two", refs.userTwo, refs.spaceOne))

	{
		resp, err := env.srv.getSummaryCordiumSecret(env.ctx, &vcordiumv1.GetSecretSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalUser)
		assert.EqualValues(t, 1, resp.TotalSpace)
	}

	insertRscStoreObject(t, env, newTstUserSecret("usec-default", now,
		cordiumv1.UserSecret_Spec_DEFAULT, "v1", refs.userOne))
	insertRscStoreObject(t, env, newTstUserSecret("usec-ssh-one", now,
		cordiumv1.UserSecret_Spec_SSH_KEY, "k1", refs.userOne))
	insertRscStoreObject(t, env, newTstUserSecret("usec-ssh-two", now,
		cordiumv1.UserSecret_Spec_SSH_KEY, "k2", refs.userTwo))

	{
		resp, err := env.srv.getSummaryCordiumUserSecret(env.ctx, &vcordiumv1.GetUserSecretSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 3, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalDefault)
		assert.EqualValues(t, 2, resp.TotalSSHKey)
		assert.EqualValues(t, 2, resp.TotalUser)
	}

	insertRscStoreObject(t, env, newTstRegion("rgn-enabled", now, true))
	insertRscStoreObject(t, env, newTstRegion("rgn-disabled", now, false))

	{
		resp, err := env.srv.getSummaryCordiumRegion(env.ctx, &vcordiumv1.GetRegionSummaryRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 1, resp.TotalEnabled)
		assert.EqualValues(t, 0, resp.TotalCountry)
	}
}

func TestCordiumSummaryComparisonWindow(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()

	insertRscStoreObject(t, env, newTstWorkspace("ws-old", now.Add(-90*time.Minute), nil,
		&cordiumv1.Workspace_Status{
			State: cordiumv1.Workspace_Status_STOPPED,
		}))
	insertRscStoreObject(t, env, newTstWorkspace("ws-new-one", now.Add(-10*time.Minute), nil,
		&cordiumv1.Workspace_Status{
			State: cordiumv1.Workspace_Status_RUNNING,
		}))
	insertRscStoreObject(t, env, newTstWorkspace("ws-new-two", now.Add(-5*time.Minute), nil,
		&cordiumv1.Workspace_Status{
			State: cordiumv1.Workspace_Status_RUNNING,
		}))

	{
		resp, err := env.srv.getSummaryCordiumWorkspace(env.ctx, &vcordiumv1.GetWorkspaceSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From:        pbutils.Timestamp(now.Add(-60 * time.Minute)),
				To:          pbutils.Timestamp(now),
				CompareFrom: pbutils.Timestamp(now.Add(-120 * time.Minute)),
				CompareTo:   pbutils.Timestamp(now.Add(-60 * time.Minute)),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.EqualValues(t, 2, resp.TotalNumber)
		assert.EqualValues(t, 2, resp.TotalRunning)
		assert.NotNil(t, resp.Previous)
		assert.EqualValues(t, 1, resp.Previous.TotalNumber)
		assert.EqualValues(t, 1, resp.Previous.TotalStopped)
		assert.Nil(t, resp.Previous.Previous)
	}

	{
		_, err := env.srv.getSummaryCordiumTemplate(env.ctx, &vcordiumv1.GetTemplateSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From: pbutils.Timestamp(now),
				To:   pbutils.Timestamp(now.Add(-time.Hour)),
			},
		})
		assert.NotNil(t, err)
	}

	{
		resp, err := env.srv.getSummaryCordiumRegion(env.ctx, &vcordiumv1.GetRegionSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				CompareFrom: pbutils.Timestamp(now.Add(-120 * time.Minute)),
				CompareTo:   pbutils.Timestamp(now.Add(-60 * time.Minute)),
			},
		})
		assert.Nil(t, err, "%+v", err)
		assert.NotNil(t, resp.Previous)
	}
}

func TestCordiumSecretDataIsNeverStored(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	srv := &srvCordium{s: env.srv}
	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertRscStoreObject(t, env, newTstCordiumSecret("space-token", now, "space-s3cr3t", refs.userOne, refs.spaceOne))
	insertRscStoreObject(t, env, newTstUserSecret("user-token", now,
		cordiumv1.UserSecret_Spec_DEFAULT, "user-s3cr3t", refs.userOne))

	{
		resp, err := srv.ListSecret(env.ctx, &vcordiumv1.ListSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "space-token", resp.Items[0].Metadata.Name)
		assert.Nil(t, resp.Items[0].Data)
	}

	{
		resp, err := srv.ListUserSecret(env.ctx, &vcordiumv1.ListUserSecretOptions{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "user-token", resp.Items[0].Metadata.Name)
		assert.Nil(t, resp.Items[0].Data)
	}

	rows, err := env.srv.db.QueryContext(env.ctx,
		`SELECT rsc_str FROM resources WHERE api = ?`, ucordiumv1.API)
	assert.Nil(t, err, "%+v", err)
	defer rows.Close()

	n := 0
	for rows.Next() {
		var stored string
		assert.Nil(t, rows.Scan(&stored))
		assert.NotContains(t, stored, "s3cr3t")
		n++
	}
	assert.Nil(t, rows.Err())
	assert.Equal(t, 2, n)
}

func TestCordiumResourceColumns(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()
	refs := newTstCordiumRefs()

	insertTstCordiumWorkspaces(t, env, refs, now)
	insertTstCordiumTemplates(t, env, refs, now)

	var wsUID, failedUID, tmplUID string
	{
		err := env.srv.db.QueryRowContext(env.ctx,
			`SELECT uid FROM resources WHERE name = 'ws-running'`).Scan(&wsUID)
		assert.Nil(t, err, "%+v", err)
		err = env.srv.db.QueryRowContext(env.ctx,
			`SELECT uid FROM resources WHERE name = 'ws-failed'`).Scan(&failedUID)
		assert.Nil(t, err, "%+v", err)
		err = env.srv.db.QueryRowContext(env.ctx,
			`SELECT uid FROM resources WHERE name = 'tmpl-ready'`).Scan(&tmplUID)
		assert.Nil(t, err, "%+v", err)
	}

	assert.Equal(t, "RUNNING", getTestColumnValue(t, env, wsUID, colStatusState))
	assert.Equal(t, refs.session.Uid, getTestColumnValue(t, env, wsUID, colSessionUID))
	assert.Equal(t, refs.spaceOne.Name, getTestColumnValue(t, env, wsUID, colSpaceName))
	assert.Equal(t, refs.template.Uid, getTestColumnValue(t, env, wsUID, colTemplateUID))
	assert.Equal(t, "USER", getTestColumnValue(t, env, wsUID, colStatusSpaceType))
	assert.Equal(t, true, getTestColumnValue(t, env, wsUID, colSpecIsEphemeral))
	assert.Equal(t, true, getTestColumnValue(t, env, wsUID, colSpecAutoStop))
	assert.EqualValues(t, 2000, getTestColumnValue(t, env, wsUID, colStatusCPUMillicores))
	assert.Equal(t, []string{"MEMBERS"}, getTestColumnList(t, env, wsUID, colStatusSharedPortModes))
	assert.NotContains(t, getTestColumnList(t, env, wsUID, colStatusRunKeys), "failure")

	assert.Equal(t, "STOPPING_REASON_ERROR", getTestColumnValue(t, env, failedUID, colStatusStoppingReason))
	assert.Equal(t, true, getTestColumnValue(t, env, failedUID, colStatusIsBuild))
	assert.Contains(t, getTestColumnList(t, env, failedUID, colStatusRunKeys), "failure")

	assert.Equal(t, "STATE_READY", getTestColumnValue(t, env, tmplUID, colStatusLatestBuildState))
	assert.Equal(t, "b2", getTestColumnValue(t, env, tmplUID, colStatusReadyBuildID))
	assert.Nil(t, getTestColumnValue(t, env, tmplUID, colStatusRunningBuildID))
	assert.Equal(t, refs.gitProvider.Uid, getTestColumnValue(t, env, tmplUID, colGitProviderUID))

	rgn := newTstRegion("rgn-enabled", now, true)
	insertRscStoreObject(t, env, rgn)
	assert.Equal(t, true, getTestColumnValue(t, env, rgn.Metadata.Uid, colStatusExtCordiumIsEnabled))
}
