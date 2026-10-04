package apiserver

import (
	"context"
	"fmt"
	"time"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/ucordiumv1"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"go.uber.org/zap"
)

func newTstCordiumLimit(cpu, memory, storage uint32) *cordiumv1.Workspace_Spec_Limit {
	return &cordiumv1.Workspace_Spec_Limit{
		Cpu:     &cordiumv1.Workspace_Spec_Limit_CPU{Millicores: cpu},
		Memory:  &cordiumv1.Workspace_Spec_Limit_Memory{Megabytes: memory},
		Storage: &cordiumv1.Workspace_Spec_Limit_Storage{Megabytes: storage},
	}
}

func genCordiumResources(ctx context.Context, octeliumC octeliumc.ClientInterface) error {

	zap.L().Debug("Generating Cordium resources")

	usrList, err := octeliumC.CoreC().ListUser(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	var users []*corev1.User
	for _, usr := range usrList.Items {
		if usr.Spec.Type == corev1.User_Spec_HUMAN && !usr.Metadata.IsSystem {
			users = append(users, usr)
		}
	}

	if len(users) < 6 {
		return nil
	}

	sessList, err := octeliumC.CoreC().ListSession(ctx, &rmetav1.ListOptions{})
	if err != nil {
		return err
	}

	rgn, err := octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
	if err != nil {
		return err
	}

	userSpaces := make(map[string]*cordiumv1.Space)

	createMembership := func(usr *corev1.User, spc *cordiumv1.Space, role cordiumv1.Membership_Spec_Role) error {
		_, err := octeliumC.CordiumC().CreateMembership(ctx, &cordiumv1.Membership{
			ApiVersion: ucordiumv1.APIVersion,
			Kind:       ucordiumv1.KindMembership,
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("%s-%s", spc.Metadata.Name, usr.Metadata.Name),
			},
			Spec: &cordiumv1.Membership_Spec{
				Role: role,
			},
			Status: &cordiumv1.Membership_Status{
				UserRef:  umetav1.GetObjectReference(usr),
				SpaceRef: umetav1.GetObjectReference(spc),
				UserInfo: &cordiumv1.Membership_Status_UserInfo{
					DisplayName: usr.Metadata.DisplayName,
				},
			},
		})
		return err
	}

	for _, usr := range users[:6] {
		spc, err := octeliumC.CordiumC().CreateSpace(ctx, &cordiumv1.Space{
			ApiVersion: ucordiumv1.APIVersion,
			Kind:       ucordiumv1.KindSpace,
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("usr-%s", usr.Metadata.Name),
			},
			Spec: &cordiumv1.Space_Spec{},
			Status: &cordiumv1.Space_Status{
				Type:    cordiumv1.Space_Status_USER,
				UserRef: umetav1.GetObjectReference(usr),
			},
		})
		if err != nil {
			return err
		}

		userSpaces[usr.Metadata.Uid] = spc

		if err := createMembership(usr, spc, cordiumv1.Membership_Spec_OWNER); err != nil {
			return err
		}
	}

	var orgSpaces []*cordiumv1.Space
	for i, name := range []string{"platform", "data-science"} {
		spc, err := octeliumC.CordiumC().CreateSpace(ctx, &cordiumv1.Space{
			ApiVersion: ucordiumv1.APIVersion,
			Kind:       ucordiumv1.KindSpace,
			Metadata: &metav1.Metadata{
				Name:        name,
				DisplayName: fmt.Sprintf("The %s team", name),
			},
			Spec: &cordiumv1.Space_Spec{
				Limit: &cordiumv1.Space_Spec_Limit{
					DefaultLimit: newTstCordiumLimit(2000, 4096, 20480),
					MaxLimit:     newTstCordiumLimit(8000, 16384, 102400),
				},
				Runtime: &cordiumv1.Space_Spec_Runtime{
					EnvVars: []*cordiumv1.Workspace_Spec_Runtime_EnvVar{
						{
							Key: "TEAM",
							Type: &cordiumv1.Workspace_Spec_Runtime_EnvVar_Value{
								Value: name,
							},
						},
					},
				},
				Authorization: &cordiumv1.Space_Spec_Authorization{
					DisableSSH: i == 1,
				},
			},
			Status: &cordiumv1.Space_Status{
				Type:    cordiumv1.Space_Status_ORGANIZATION,
				UserRef: umetav1.GetObjectReference(users[i]),
			},
		})
		if err != nil {
			return err
		}

		orgSpaces = append(orgSpaces, spc)

		for j, usr := range users {
			role := cordiumv1.Membership_Spec_USER
			switch {
			case j == i:
				role = cordiumv1.Membership_Spec_OWNER
			case j == i+2:
				role = cordiumv1.Membership_Spec_ADMIN
			case j > 8:
				continue
			}

			if err := createMembership(usr, spc, role); err != nil {
				return err
			}
		}
	}

	gitProviders := []*cordiumv1.GitProvider{
		{
			Metadata: &metav1.Metadata{Name: "github"},
			Spec: &cordiumv1.GitProvider_Spec{
				Type: &cordiumv1.GitProvider_Spec_Github_{
					Github: &cordiumv1.GitProvider_Spec_Github{
						ClientID: "Iv1.8a61f9b3a7aba766",
						ClientSecret: &cordiumv1.GitProvider_Spec_Github_ClientSecret{
							Type: &cordiumv1.GitProvider_Spec_Github_ClientSecret_FromSecret{
								FromSecret: "github-client-secret",
							},
						},
						Scopes: []string{"repo", "read:org"},
					},
				},
			},
			Status: &cordiumv1.GitProvider_Status{
				UserRef:  umetav1.GetObjectReference(users[0]),
				SpaceRef: umetav1.GetObjectReference(orgSpaces[0]),
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "gitlab"},
			Spec: &cordiumv1.GitProvider_Spec{
				Type: &cordiumv1.GitProvider_Spec_Gitlab_{
					Gitlab: &cordiumv1.GitProvider_Spec_Gitlab{
						ClientID: "a3f1c0d2e4b5",
						Scopes:   []string{"read_repository"},
					},
				},
			},
			Status: &cordiumv1.GitProvider_Status{
				UserRef:  umetav1.GetObjectReference(users[1]),
				SpaceRef: umetav1.GetObjectReference(orgSpaces[1]),
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "gitea"},
			Spec: &cordiumv1.GitProvider_Spec{
				Type: &cordiumv1.GitProvider_Spec_Oauth2{
					Oauth2: &cordiumv1.GitProvider_Spec_OAuth2{
						ClientID: "gitea-client",
						AuthURL:  "https://git.example.com/login/oauth/authorize",
						TokenURL: "https://git.example.com/login/oauth/access_token",
						Scopes:   []string{"repo"},
					},
				},
			},
			Status: &cordiumv1.GitProvider_Status{
				UserRef:  umetav1.GetObjectReference(users[2]),
				SpaceRef: umetav1.GetObjectReference(userSpaces[users[2].Metadata.Uid]),
			},
		},
	}

	for i, gp := range gitProviders {
		gp.ApiVersion = ucordiumv1.APIVersion
		gp.Kind = ucordiumv1.KindGitProvider
		gitProviders[i], err = octeliumC.CordiumC().CreateGitProvider(ctx, gp)
		if err != nil {
			return err
		}
	}

	now := time.Now()
	newBuild := func(id string, state cordiumv1.Template_Status_BuildInfo_Build_State,
		startedAgo time.Duration, isCanceled bool, failure *cordiumv1.Workspace_Status_Failure) *cordiumv1.Template_Status_BuildInfo_Build {
		ret := &cordiumv1.Template_Status_BuildInfo_Build{
			Id:         id,
			State:      state,
			StartedAt:  pbutils.Timestamp(now.Add(-startedAgo)),
			IsCanceled: isCanceled,
			Failure:    failure,
		}
		if state != cordiumv1.Template_Status_BuildInfo_Build_STATE_RUNNING {
			ret.DoneAt = pbutils.Timestamp(now.Add(-startedAgo).Add(4 * time.Minute))
		}

		return ret
	}

	templates := []*cordiumv1.Template{
		{
			Metadata: &metav1.Metadata{Name: "go-dev", DisplayName: "Go development"},
			Spec: &cordiumv1.Template_Spec{
				Image: &cordiumv1.Workspace_Spec_Image{
					Type: &cordiumv1.Workspace_Spec_Image_Registry_{
						Registry: &cordiumv1.Workspace_Spec_Image_Registry{
							Url: "ghcr.io/octelium/devcontainers/go:1.26",
						},
					},
				},
				Repository: &cordiumv1.Workspace_Spec_Repository{
					Url: "https://github.com/octelium/octelium",
				},
				Limit:       newTstCordiumLimit(2000, 4096, 20480),
				GitProvider: gitProviders[0].Metadata.Name,
			},
			Status: &cordiumv1.Template_Status{
				SpaceRef:       umetav1.GetObjectReference(orgSpaces[0]),
				UserRef:        umetav1.GetObjectReference(users[0]),
				GitProviderRef: umetav1.GetObjectReference(gitProviders[0]),
				BuildInfo: &cordiumv1.Template_Status_BuildInfo{
					Builds: []*cordiumv1.Template_Status_BuildInfo_Build{
						newBuild("b7k2q1", cordiumv1.Template_Status_BuildInfo_Build_STATE_READY, 2*time.Hour, false, nil),
						newBuild("x9m3p0", cordiumv1.Template_Status_BuildInfo_Build_STATE_FAILED, 26*time.Hour, false,
							&cordiumv1.Workspace_Status_Failure{
								Message: "go: module lookup disabled by GOPROXY=off",
								Type: &cordiumv1.Workspace_Status_Failure_ImageBuild_{
									ImageBuild: &cordiumv1.Workspace_Status_Failure_ImageBuild{},
								},
							}),
					},
					CurrentReadyBuildID: "b7k2q1",
				},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "python-ml", DisplayName: "Python ML notebooks"},
			Spec: &cordiumv1.Template_Spec{
				Image: &cordiumv1.Workspace_Spec_Image{
					Type: &cordiumv1.Workspace_Spec_Image_Git_{
						Git: &cordiumv1.Workspace_Spec_Image_Git{
							Url:        "https://gitlab.example.com/data/ml-image",
							Dockerfile: "Dockerfile",
						},
					},
				},
				Limit: newTstCordiumLimit(4000, 16384, 51200),
			},
			Status: &cordiumv1.Template_Status{
				SpaceRef: umetav1.GetObjectReference(orgSpaces[1]),
				UserRef:  umetav1.GetObjectReference(users[1]),
				BuildInfo: &cordiumv1.Template_Status_BuildInfo{
					Builds: []*cordiumv1.Template_Status_BuildInfo_Build{
						newBuild("r4n8t2", cordiumv1.Template_Status_BuildInfo_Build_STATE_RUNNING, 5*time.Minute, false, nil),
						newBuild("c1v6z3", cordiumv1.Template_Status_BuildInfo_Build_STATE_READY, 50*time.Hour, false, nil),
						newBuild("k0s5w9", cordiumv1.Template_Status_BuildInfo_Build_STATE_FAILED, 72*time.Hour, true, nil),
					},
					CurrentReadyBuildID:   "c1v6z3",
					CurrentRunningBuildID: "r4n8t2",
				},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "node-web"},
			Spec: &cordiumv1.Template_Spec{
				Image: &cordiumv1.Workspace_Spec_Image{
					Type: &cordiumv1.Workspace_Spec_Image_Repository_{
						Repository: &cordiumv1.Workspace_Spec_Image_Repository{
							Type: &cordiumv1.Workspace_Spec_Image_Repository_Devcontainer_{
								Devcontainer: &cordiumv1.Workspace_Spec_Image_Repository_Devcontainer{},
							},
						},
					},
				},
				Repository: &cordiumv1.Workspace_Spec_Repository{
					Url: "https://git.example.com/web/storefront",
				},
			},
			Status: &cordiumv1.Template_Status{
				SpaceRef: umetav1.GetObjectReference(userSpaces[users[2].Metadata.Uid]),
				UserRef:  umetav1.GetObjectReference(users[2]),
				BuildInfo: &cordiumv1.Template_Status_BuildInfo{
					Builds: []*cordiumv1.Template_Status_BuildInfo_Build{
						newBuild("h2d7f4", cordiumv1.Template_Status_BuildInfo_Build_STATE_FAILED, 40*time.Minute, false,
							&cordiumv1.Workspace_Status_Failure{
								Message: "devcontainer.json: unknown feature",
								Type: &cordiumv1.Workspace_Status_Failure_ImageBuild_{
									ImageBuild: &cordiumv1.Workspace_Status_Failure_ImageBuild{},
								},
							}),
					},
				},
			},
		},
		{
			Metadata: &metav1.Metadata{Name: "rust-dev"},
			Spec: &cordiumv1.Template_Spec{
				Image: &cordiumv1.Workspace_Spec_Image{
					Type: &cordiumv1.Workspace_Spec_Image_Registry_{
						Registry: &cordiumv1.Workspace_Spec_Image_Registry{
							Url: "docker.io/library/rust:1.90",
						},
					},
				},
			},
			Status: &cordiumv1.Template_Status{
				SpaceRef: umetav1.GetObjectReference(orgSpaces[0]),
				UserRef:  umetav1.GetObjectReference(users[3]),
			},
		},
	}

	for i, tmpl := range templates {
		tmpl.ApiVersion = ucordiumv1.APIVersion
		tmpl.Kind = ucordiumv1.KindTemplate
		templates[i], err = octeliumC.CordiumC().CreateTemplate(ctx, tmpl)
		if err != nil {
			return err
		}
	}

	getSessionRef := func(usr *corev1.User) *metav1.ObjectReference {
		for _, sess := range sessList.Items {
			if sess.Status.UserRef != nil && sess.Status.UserRef.Uid == usr.Metadata.Uid {
				return umetav1.GetObjectReference(sess)
			}
		}
		return nil
	}

	states := []cordiumv1.Workspace_Status_State{
		cordiumv1.Workspace_Status_RUNNING,
		cordiumv1.Workspace_Status_RUNNING,
		cordiumv1.Workspace_Status_RUNNING,
		cordiumv1.Workspace_Status_RUNNING,
		cordiumv1.Workspace_Status_RUNNING,
		cordiumv1.Workspace_Status_INITIALIZING,
		cordiumv1.Workspace_Status_PULLING_IMAGE,
		cordiumv1.Workspace_Status_BUILDING_IMAGE,
		cordiumv1.Workspace_Status_STARTING_RUNTIME,
		cordiumv1.Workspace_Status_PREPARING,
		cordiumv1.Workspace_Status_STOPPING,
		cordiumv1.Workspace_Status_STOPPED,
		cordiumv1.Workspace_Status_STOPPED,
		cordiumv1.Workspace_Status_STOPPED,
		cordiumv1.Workspace_Status_STOPPED,
		cordiumv1.Workspace_Status_STOPPED,
	}

	failures := []*cordiumv1.Workspace_Status_Failure{
		{
			Message: "manifest unknown: ghcr.io/octelium/devcontainers/go:1.27",
			Type: &cordiumv1.Workspace_Status_Failure_ImagePull_{
				ImagePull: &cordiumv1.Workspace_Status_Failure_ImagePull{},
			},
		},
		{
			Message: "npm ci exited with an error",
			Type: &cordiumv1.Workspace_Status_Failure_Task_{
				Task: &cordiumv1.Workspace_Status_Failure_Task{
					Name:     "install-deps",
					ExitCode: 1,
				},
			},
		},
	}

	for i, state := range states {
		usr := users[i%len(users)]
		tmpl := templates[i%len(templates)]

		spc := userSpaces[usr.Metadata.Uid]
		spaceType := cordiumv1.Space_Status_USER
		if spc == nil || i%3 == 0 {
			spc = orgSpaces[i%len(orgSpaces)]
			spaceType = cordiumv1.Space_Status_ORGANIZATION
		}

		ws := &cordiumv1.Workspace{
			ApiVersion: ucordiumv1.APIVersion,
			Kind:       ucordiumv1.KindWorkspace,
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("%s-%s", tmpl.Metadata.Name, utilrand.GetRandomStringCanonical(5)),
			},
			Spec: &cordiumv1.Workspace_Spec{
				Image:       tmpl.Spec.Image,
				Repository:  tmpl.Spec.Repository,
				IsEphemeral: i%4 == 1,
				Runtime: &cordiumv1.Workspace_Spec_Runtime{
					AutoStop: i%5 == 2,
				},
				Applications: []*cordiumv1.Workspace_Spec_Application{
					{
						Name: "web",
						Port: 3000,
					},
				},
			},
			Status: &cordiumv1.Workspace_Status{
				State:             state,
				UserRef:           umetav1.GetObjectReference(usr),
				SpaceRef:          umetav1.GetObjectReference(spc),
				SpaceType:         spaceType,
				TemplateRef:       umetav1.GetObjectReference(tmpl),
				CurrentStateSetAt: pbutils.Timestamp(now.Add(-time.Duration(i*7) * time.Minute)),
				LastInitializedAt: pbutils.Timestamp(now.Add(-time.Duration(i*9+5) * time.Minute)),
				LastActivityAt:    pbutils.Timestamp(now.Add(-time.Duration(i*3) * time.Minute)),
				SuccessfulRuns:    uint32(i % 7),
				IsBuild:           i == 8,
				Run: &cordiumv1.Workspace_Status_Run{
					Id:            utilrand.GetRandomStringCanonical(6),
					InitializedAt: pbutils.Timestamp(now.Add(-time.Duration(i*9+5) * time.Minute)),
				},
			},
		}

		if state != cordiumv1.Workspace_Status_STOPPED {
			ws.Status.RegionRef = umetav1.GetObjectReference(rgn)
			ws.Status.LastRegionRef = ws.Status.RegionRef
			ws.Status.SessionRef = getSessionRef(usr)
			ws.Status.Hostname = fmt.Sprintf("%s.ws.example.com", ws.Metadata.Name)
			ws.Status.Limit = newTstCordiumLimit(uint32(1000+(i%4)*1000), uint32(2048+(i%3)*2048), 20480)
		} else {
			ws.Status.LastRegionRef = umetav1.GetObjectReference(rgn)
			ws.Status.LastStoppedAt = ws.Status.CurrentStateSetAt
			ws.Status.StoppingReason = cordiumv1.Workspace_Status_STOPPING_REASON_API
			ws.Status.Run.StoppedAt = ws.Status.CurrentStateSetAt
			ws.Status.Limit = newTstCordiumLimit(2000, 4096, 20480)
		}

		if state == cordiumv1.Workspace_Status_RUNNING {
			ws.Status.LastRunningAt = ws.Status.CurrentStateSetAt
			switch i % 3 {
			case 0:
				ws.Status.SharedPorts = []*cordiumv1.Workspace_Status_SharedPort{
					{Mode: cordiumv1.Workspace_Status_SharedPort_MEMBERS, ApplicationName: "web"},
				}
			case 1:
				ws.Status.SharedPorts = []*cordiumv1.Workspace_Status_SharedPort{
					{Mode: cordiumv1.Workspace_Status_SharedPort_ALL, ApplicationName: "web"},
				}
			}
		}

		if i >= 14 {
			ws.Status.StoppingReason = cordiumv1.Workspace_Status_STOPPING_REASON_ERROR
			ws.Status.Run.Failure = failures[i%len(failures)]
			ws.Status.Failure = ws.Status.Run.Failure
		}

		if _, err := octeliumC.CordiumC().CreateWorkspace(ctx, ws); err != nil {
			return err
		}
	}

	for i, spc := range orgSpaces {
		if _, err := octeliumC.CordiumC().CreateSecret(ctx, &cordiumv1.Secret{
			ApiVersion: ucordiumv1.APIVersion,
			Kind:       ucordiumv1.KindSecret,
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("%s-registry-token", spc.Metadata.Name),
			},
			Spec: &cordiumv1.Secret_Spec{},
			Status: &cordiumv1.Secret_Status{
				SpaceRef: umetav1.GetObjectReference(spc),
				UserRef:  umetav1.GetObjectReference(users[i]),
			},
			Data: &cordiumv1.Secret_Data{
				Type: &cordiumv1.Secret_Data_Value{
					Value: utilrand.GetRandomString(32),
				},
			},
		}); err != nil {
			return err
		}
	}

	for i, usr := range users[:4] {
		usec := &cordiumv1.UserSecret{
			ApiVersion: ucordiumv1.APIVersion,
			Kind:       ucordiumv1.KindUserSecret,
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("%s-secret", usr.Metadata.Name),
			},
			Spec: &cordiumv1.UserSecret_Spec{},
			Status: &cordiumv1.UserSecret_Status{
				UserRef: umetav1.GetObjectReference(usr),
			},
			Data: &cordiumv1.UserSecret_Data{
				Type: &cordiumv1.UserSecret_Data_Value{
					Value: utilrand.GetRandomString(32),
				},
			},
		}

		if i%2 == 0 {
			usec.Metadata.Name = fmt.Sprintf("%s-ssh", usr.Metadata.Name)
			usec.Spec.Type = cordiumv1.UserSecret_Spec_SSH_KEY
			usec.Status.Details = &cordiumv1.UserSecret_Status_SshKey{
				SshKey: &cordiumv1.UserSecret_Status_SSHKey{
					PublicKey: fmt.Sprintf("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI%s %s",
						utilrand.GetRandomStringCanonical(32), usr.Metadata.Name),
				},
			}
		}

		if _, err := octeliumC.CordiumC().CreateUserSecret(ctx, usec); err != nil {
			return err
		}
	}

	zap.L().Debug("Done generating Cordium resources")

	return nil
}
