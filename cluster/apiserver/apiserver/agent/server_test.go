// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/admin"
	"github.com/octelium/octelium/cluster/apiserver/apiserver/user"
	"github.com/octelium/octelium/cluster/common/tests/tstuser"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func tstSetCordiumInstalled(ctx context.Context, t *testing.T, srv *Server, installed bool) {
	rgn, err := srv.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{Name: "default"})
	assert.Nil(t, err, "%+v", err)
	if rgn.Status == nil {
		rgn.Status = &corev1.Region_Status{}
	}
	if rgn.Status.VersionInfoMap == nil {
		rgn.Status.VersionInfoMap = make(map[string]*corev1.Region_Status_VersionInfo)
	}

	if installed {
		rgn.Status.VersionInfoMap[versionInfoKeyCordium] = &corev1.Region_Status_VersionInfo{
			Version: "1.0.0",
			SetAt:   pbutils.Now(),
		}
	} else {
		delete(rgn.Status.VersionInfoMap, versionInfoKeyCordium)
	}

	_, err = srv.octeliumC.CoreC().UpdateRegion(ctx, rgn)
	assert.Nil(t, err, "%+v", err)
}

func tstSetAgentConfig(ctx context.Context, t *testing.T, srv *Server, cfg *enterprisev1.ClusterConfig_Spec_Agent) {
	cc, err := srv.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err, "%+v", err)
	if cc.Spec == nil {
		cc.Spec = &enterprisev1.ClusterConfig_Spec{}
	}
	cc.Spec.Agent = cfg

	_, err = srv.octeliumC.EnterpriseC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err, "%+v", err)
}

func TestAgent(t *testing.T) {
	t.Setenv("OCTELIUM_DEV", "true")
	oldGitBranch := ldflags.GitBranch
	ldflags.GitBranch = "dev"
	t.Cleanup(func() {
		ldflags.GitBranch = oldGitBranch
	})

	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	})
	usrSrv := user.NewServer(tst.C.OcteliumC)
	srv := NewServer(tst.C.OcteliumC)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	{
		_, err := srv.GetAgent(usr.Ctx(), nil)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		_, err = srv.InitializeAgent(usr.Ctx(), nil)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)

		_, err = srv.CreateAgentWorkspace(usr.Ctx(), nil)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	tstSetCordiumInstalled(ctx, t, srv, false)

	{
		_, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)

		_, err = srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
	}

	tstSetCordiumInstalled(ctx, t, srv, true)

	{
		res, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_NOT_INITIALIZED, res.State)
		assert.Nil(t, res.SpaceRef)
		assert.Len(t, res.Workspaces, 0)

		_, err = srv.CreateAgentWorkspace(usr.Ctx(), &enterprisev1.CreateAgentWorkspaceRequest{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
	}

	var primary *cordiumv1.Workspace

	{
		res, err := srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		assert.Equal(t, fmt.Sprintf("octelium.%s", usr.Usr.Metadata.Name), res.SpaceRef.Name)
		assert.Equal(t, fmt.Sprintf("default.octelium.%s", usr.Usr.Metadata.Name), res.TemplateRef.Name)
		assert.Len(t, res.Workspaces, 1)
		assert.Equal(t, enterprisev1.Agent_Workspace_PRIMARY, res.Workspaces[0].Type)
		assert.Equal(t, "", res.Workspaces[0].Url)

		primary = res.Workspaces[0].Workspace
		assert.Equal(t, cordiumv1.Workspace_Status_STOPPED, primary.Status.State)
		assert.Equal(t, usr.Usr.Metadata.Uid, primary.Status.UserRef.Uid)
		assert.Equal(t, res.SpaceRef.Uid, primary.Status.SpaceRef.Uid)
		assert.Equal(t, res.TemplateRef.Uid, primary.Status.TemplateRef.Uid)
		assert.Equal(t, cordiumv1.Space_Status_USER, primary.Status.SpaceType)
		assert.Equal(t, workspaceTypePrimary, primary.Metadata.SystemLabels[systemLabelAgent])
		assert.False(t, primary.Spec.IsEphemeral)
		assert.Len(t, primary.Spec.Applications, 1)
		assert.True(t, primary.Spec.Applications[0].IsDefault)
		assert.Equal(t, int32(applicationPort), primary.Spec.Applications[0].Port)

		spc, err := srv.octeliumC.CordiumC().GetSpace(ctx, &rmetav1.GetOptions{Uid: res.SpaceRef.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, spc.Metadata.IsSystem)
		assert.Equal(t, "true", spc.Metadata.SystemLabels[systemLabelAgent])
		assert.Equal(t, "user", spc.Metadata.SystemLabels["type"])
		assert.Equal(t, cordiumv1.Space_Status_USER, spc.Status.Type)
		assert.Equal(t, usr.Usr.Metadata.Uid, spc.Status.UserRef.Uid)

		mem, err := srv.octeliumC.CordiumC().GetMembership(ctx, &rmetav1.GetOptions{
			Name: fmt.Sprintf("%s.%s", usr.Usr.Metadata.Name, spc.Metadata.Name),
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, cordiumv1.Membership_Spec_OWNER, mem.Spec.Role)
		assert.Equal(t, spc.Metadata.Uid, mem.Status.SpaceRef.Uid)
		assert.Equal(t, usr.Usr.Metadata.Uid, mem.Status.UserRef.Uid)

		tmpl, err := srv.octeliumC.CordiumC().GetTemplate(ctx, &rmetav1.GetOptions{Uid: res.TemplateRef.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.True(t, tmpl.Metadata.IsSystem)
		assert.Equal(t, spc.Metadata.Uid, tmpl.Status.SpaceRef.Uid)
		assert.Len(t, tmpl.Spec.Runtime.Tasks, 2)
		assert.Equal(t, fmt.Sprintf("exec npx --yes --prefer-online %s@%s serve", consoleAgentPackage, defaultConsoleAgentVersion),
			tmpl.Spec.Runtime.Tasks[1].Run)
		assert.Len(t, tmpl.Spec.Runtime.EnvVars, 0)
	}

	{
		res, err := srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		assert.Len(t, res.Workspaces, 1)
		assert.Equal(t, primary.Metadata.Uid, res.Workspaces[0].Workspace.Metadata.Uid)
	}

	{
		res, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		assert.Len(t, res.Workspaces, 1)
	}

	{
		tstSetAgentConfig(ctx, t, srv, &enterprisev1.ClusterConfig_Spec_Agent{
			Version: "0.2.0",
			Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{
				Service: "llm.default",
				Model:   "gpt-5.1",
			},
		})

		res, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_OUTDATED, res.State)

		res, err = srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		assert.Len(t, res.Workspaces, 1)

		tmpl, err := srv.octeliumC.CordiumC().GetTemplate(ctx, &rmetav1.GetOptions{Uid: res.TemplateRef.Uid})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, fmt.Sprintf("exec npx --yes --prefer-online %s@0.2.0 serve", consoleAgentPackage), tmpl.Spec.Runtime.Tasks[1].Run)
		assert.Len(t, tmpl.Spec.Runtime.EnvVars, 1)
		assert.Equal(t, configEnvVar, tmpl.Spec.Runtime.EnvVars[0].Key)

		agentCfg := make(map[string]any)
		assert.Nil(t, json.Unmarshal([]byte(tmpl.Spec.Runtime.EnvVars[0].GetValue()), &agentCfg))
		assert.Equal(t, map[string]any{
			"provider": "octelium",
			"service":  "llm.default",
			"model":    "gpt-5.1",
		}, agentCfg["llm"])
	}

	{
		_, err := srv.CreateAgentWorkspace(usr.Ctx(), &enterprisev1.CreateAgentWorkspaceRequest{
			DisplayName: strings.Repeat("a", maxDisplayNameLen+1),
		})
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
	}

	{
		ws, err := srv.CreateAgentWorkspace(usr.Ctx(), &enterprisev1.CreateAgentWorkspaceRequest{
			DisplayName: "Fresh",
			IsEphemeral: true,
		})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_Workspace_ADDITIONAL, ws.Type)
		assert.Equal(t, "Fresh", ws.Workspace.Metadata.DisplayName)
		assert.True(t, ws.Workspace.Spec.IsEphemeral)
		assert.Equal(t, workspaceTypeAdditional, ws.Workspace.Metadata.SystemLabels[systemLabelAgent])
		assert.NotEqual(t, primary.Metadata.Name, ws.Workspace.Metadata.Name)

		res, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Len(t, res.Workspaces, 2)
		assert.Equal(t, enterprisev1.Agent_Workspace_PRIMARY, res.Workspaces[0].Type)
		assert.Equal(t, primary.Metadata.Uid, res.Workspaces[0].Workspace.Metadata.Uid)
		assert.Equal(t, enterprisev1.Agent_Workspace_ADDITIONAL, res.Workspaces[1].Type)
		assert.Equal(t, ws.Workspace.Metadata.Uid, res.Workspaces[1].Workspace.Metadata.Uid)
	}

	{
		ws, err := srv.octeliumC.CordiumC().GetWorkspace(ctx, &rmetav1.GetOptions{Uid: primary.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)
		ws.Status.Hostname = fmt.Sprintf("%s.cordium.example.com", ws.Metadata.Name)
		_, err = srv.octeliumC.CordiumC().UpdateWorkspace(ctx, ws)
		assert.Nil(t, err, "%+v", err)

		res, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, fmt.Sprintf("https://%s.cordium.example.com", ws.Metadata.Name), res.Workspaces[0].Url)
	}

	{
		_, err := srv.octeliumC.CordiumC().DeleteWorkspace(ctx, &rmetav1.DeleteOptions{Uid: primary.Metadata.Uid})
		assert.Nil(t, err, "%+v", err)

		res, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_OUTDATED, res.State)
		assert.Len(t, res.Workspaces, 1)
		assert.Equal(t, enterprisev1.Agent_Workspace_ADDITIONAL, res.Workspaces[0].Type)

		res, err = srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		assert.Len(t, res.Workspaces, 2)
		assert.Equal(t, enterprisev1.Agent_Workspace_PRIMARY, res.Workspaces[0].Type)
		assert.NotEqual(t, primary.Metadata.Uid, res.Workspaces[0].Workspace.Metadata.Uid)
	}

	{
		usr2, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
		assert.Nil(t, err)

		res, err := srv.GetAgent(usr2.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, enterprisev1.Agent_NOT_INITIALIZED, res.State)
	}

	{
		usr3, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
		assert.Nil(t, err)

		_, err = srv.octeliumC.CordiumC().CreateSpace(ctx, &cordiumv1.Space{
			Metadata: &metav1.Metadata{
				Name: fmt.Sprintf("octelium.%s", usr3.Usr.Metadata.Name),
			},
			Spec: &cordiumv1.Space_Spec{},
			Status: &cordiumv1.Space_Status{
				UserRef: umetav1.GetObjectReference(usr3.Usr),
				Type:    cordiumv1.Space_Status_USER,
			},
		})
		assert.Nil(t, err, "%+v", err)

		_, err = srv.GetAgent(usr3.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)

		_, err = srv.InitializeAgent(usr3.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.True(t, grpcerr.AlreadyExists(err), "%+v", err)
	}

	{
		tstSetAgentConfig(ctx, t, srv, &enterprisev1.ClusterConfig_Spec_Agent{
			IsDisabled: true,
		})

		_, err := srv.GetAgent(usr.Ctx(), &enterprisev1.GetAgentRequest{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)

		_, err = srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)

		_, err = srv.CreateAgentWorkspace(usr.Ctx(), &enterprisev1.CreateAgentWorkspaceRequest{})
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
	}
}

func TestAgentMaxWorkspaces(t *testing.T) {
	ctx := context.Background()

	tst, err := tests.Initialize(nil)
	assert.Nil(t, err)
	t.Cleanup(func() {
		tst.Destroy()
	})

	adminSrv := admin.NewServer(&admin.Opts{
		OcteliumC:  tst.C.OcteliumC,
		IsEmbedded: true,
	})
	usrSrv := user.NewServer(tst.C.OcteliumC)
	srv := NewServer(tst.C.OcteliumC)
	tstSetCordiumInstalled(ctx, t, srv, true)

	cc, err := srv.octeliumC.CordiumV1Utils().GetClusterConfig(ctx)
	assert.Nil(t, err, "%+v", err)
	cc.Spec = &cordiumv1.ClusterConfig_Spec{
		Workspace: &cordiumv1.ClusterConfig_Spec_Workspace{
			Limit: &cordiumv1.ClusterConfig_Spec_Workspace_Limit{
				MaxPerUser: 2,
			},
		},
	}
	_, err = srv.octeliumC.CordiumC().UpdateClusterConfig(ctx, cc)
	assert.Nil(t, err, "%+v", err)

	usr, err := tstuser.NewUser(tst.C.OcteliumC, adminSrv, usrSrv, nil)
	assert.Nil(t, err)

	_, err = srv.InitializeAgent(usr.Ctx(), &enterprisev1.InitializeAgentRequest{})
	assert.Nil(t, err, "%+v", err)

	_, err = srv.CreateAgentWorkspace(usr.Ctx(), &enterprisev1.CreateAgentWorkspaceRequest{
		DisplayName: utilrand.GetRandomStringCanonical(8),
	})
	assert.Nil(t, err, "%+v", err)

	_, err = srv.CreateAgentWorkspace(usr.Ctx(), &enterprisev1.CreateAgentWorkspaceRequest{})
	assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
}

func TestTemplateSpec(t *testing.T) {
	t.Setenv("OCTELIUM_DEV", "true")
	oldGitBranch := ldflags.GitBranch
	ldflags.GitBranch = "dev"
	t.Cleanup(func() {
		ldflags.GitBranch = oldGitBranch
	})

	{
		spec, err := getTemplateSpec(nil)
		assert.Nil(t, err, "%+v", err)
		assert.Nil(t, spec.Image)
		assert.Nil(t, spec.Limit)
		assert.Len(t, spec.Runtime.EnvVars, 0)
		assert.Len(t, spec.Runtime.Tasks, 2)
		assert.Equal(t, cordiumv1.Workspace_Spec_Runtime_Task_ON_CREATE, spec.Runtime.Tasks[0].Type)
		assert.True(t, spec.Runtime.Tasks[0].RunAsRoot)
		assert.Equal(t, cordiumv1.Workspace_Spec_Runtime_Task_POST_START, spec.Runtime.Tasks[1].Type)
		assert.True(t, spec.Runtime.Tasks[1].IsBackground)
		assert.Equal(t, fmt.Sprintf("exec npx --yes --prefer-online %s@%s serve", consoleAgentPackage, defaultConsoleAgentVersion),
			spec.Runtime.Tasks[1].Run)
	}

	{
		cfg := &enterprisev1.ClusterConfig_Spec_Agent{
			Version: "1.2.3-beta.1",
			Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{
				Service: "llm",
			},
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Registry_{
					Registry: &cordiumv1.Workspace_Spec_Image_Registry{
						Url: "ghcr.io/octelium/console-agent:latest",
					},
				},
			},
			Limit: &cordiumv1.Workspace_Spec_Limit{
				Memory: &cordiumv1.Workspace_Spec_Limit_Memory{
					Megabytes: 4000,
				},
			},
			Config: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"llm": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"thinkingLevel": structpb.NewStringValue("high"),
						},
					}),
					"approvals": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"bash": structpb.NewBoolValue(true),
						},
					}),
				},
			},
		}

		spec, err := getTemplateSpec(cfg)
		assert.Nil(t, err, "%+v", err)
		assert.True(t, pbutils.IsEqual(cfg.Image, spec.Image))
		assert.True(t, pbutils.IsEqual(cfg.Limit, spec.Limit))
		assert.Equal(t, fmt.Sprintf("exec npx --yes --prefer-online %s@1.2.3-beta.1 serve", consoleAgentPackage), spec.Runtime.Tasks[1].Run)
		assert.Len(t, spec.Runtime.EnvVars, 1)

		agentCfg := make(map[string]any)
		assert.Nil(t, json.Unmarshal([]byte(spec.Runtime.EnvVars[0].GetValue()), &agentCfg))
		assert.Equal(t, map[string]any{
			"llm": map[string]any{
				"provider":      "octelium",
				"service":       "llm",
				"thinkingLevel": "high",
			},
			"approvals": map[string]any{
				"bash": true,
			},
		}, agentCfg)

		spec2, err := getTemplateSpec(cfg)
		assert.Nil(t, err, "%+v", err)
		assert.True(t, pbutils.IsEqual(spec, spec2))
	}
}

func TestTemplateSpecVersion(t *testing.T) {
	oldGitBranch := ldflags.GitBranch
	oldGitTag := ldflags.GitTag
	oldMode := ldflags.Mode
	ldflags.Mode = ""
	t.Cleanup(func() {
		ldflags.GitBranch = oldGitBranch
		ldflags.GitTag = oldGitTag
		ldflags.Mode = oldMode
	})

	tests := []struct {
		name       string
		branch     string
		gitTag     string
		dev        string
		production string
		version    string
		want       string
		isInvalid  bool
	}{
		{name: "local", want: "dev"},
		{name: "main", branch: "main", want: "main"},
		{name: "dev", branch: "dev", want: "dev"},
		{name: "feature", branch: "b-feature", want: "b-feature"},
		{name: "underscore", branch: "b-feature_1", want: "b-feature_1"},
		{name: "release", branch: "main", gitTag: "v1.2.3", want: "latest"},
		{name: "production", branch: "dev", production: "true", want: "latest"},
		{name: "production override", branch: "dev", production: "true", version: "b-feature", want: "latest"},
		{name: "production pin", branch: "dev", production: "true", version: "1.2.3", want: "latest"},
		{name: "development release", branch: "b-feature", gitTag: "v1.2.3", dev: "true", want: "b-feature"},
		{name: "development override", branch: "b-feature", version: "canary", want: "canary"},
		{name: "development pin", branch: "dev", version: "1.2.3", want: "1.2.3"},
		{name: "invalid branch", branch: "dev; touch /tmp/agent", isInvalid: true},
	}
	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			t.Setenv("OCTELIUM_DEV", tst.dev)
			t.Setenv("OCTELIUM_PRODUCTION", tst.production)
			ldflags.GitBranch = tst.branch
			ldflags.GitTag = tst.gitTag
			spec, err := getTemplateSpec(&enterprisev1.ClusterConfig_Spec_Agent{
				Version: tst.version,
			})
			if tst.isInvalid {
				assert.True(t, grpcerr.IsInvalidArg(err), "%+v", err)
				return
			}
			assert.Nil(t, err, "%+v", err)
			if !assert.NotNil(t, spec) {
				return
			}
			assert.Equal(t, fmt.Sprintf("exec npx --yes --prefer-online %s@%s serve", consoleAgentPackage, tst.want),
				spec.Runtime.Tasks[1].Run)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	validCfgs := []*enterprisev1.ClusterConfig_Spec_Agent{
		nil,
		{},
		{Version: "0.1.0"},
		{Version: "latest"},
		{Version: "dev"},
		{Version: "main"},
		{Version: "b-feature_1"},
		{Version: "1.0.0-rc.1+build.5"},
		{Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{Service: "llm"}},
		{Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{Service: "llm.default", Model: "claude-sonnet-4-5"}},
		{
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Dockerfile_{
					Dockerfile: &cordiumv1.Workspace_Spec_Image_Dockerfile{
						Type: &cordiumv1.Workspace_Spec_Image_Dockerfile_Inline{
							Inline: "FROM node:24",
						},
					},
				},
			},
		},
		{
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Git_{
					Git: &cordiumv1.Workspace_Spec_Image_Git{
						Url: "https://github.com/octelium/images",
					},
				},
			},
		},
		{
			Limit: &cordiumv1.Workspace_Spec_Limit{
				Cpu: &cordiumv1.Workspace_Spec_Limit_CPU{
					Millicores: 4000,
				},
			},
		},
	}

	for _, cfg := range validCfgs {
		assert.Nil(t, ValidateConfig(cfg), "%+v", cfg)
	}

	invalidCfgs := []*enterprisev1.ClusterConfig_Spec_Agent{
		{Version: "0.1.0; rm -rf /"},
		{Version: "-1"},
		{Version: "0.1.0 serve"},
		{Version: strings.Repeat("1", 200)},
		{Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{Service: "Invalid Name"}},
		{Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{Service: "a.b.c"}},
		{Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{Model: "gpt\n5"}},
		{Llm: &enterprisev1.ClusterConfig_Spec_Agent_LLM{Model: strings.Repeat("a", maxModelLen+1)}},
		{Image: &cordiumv1.Workspace_Spec_Image{}},
		{
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Registry_{
					Registry: &cordiumv1.Workspace_Spec_Image_Registry{},
				},
			},
		},
		{
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Registry_{
					Registry: &cordiumv1.Workspace_Spec_Image_Registry{
						Url:            "registry.example.com/agent:latest",
						Authentication: &cordiumv1.Workspace_Spec_Image_Registry_Authentication{},
					},
				},
			},
		},
		{
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Dockerfile_{
					Dockerfile: &cordiumv1.Workspace_Spec_Image_Dockerfile{
						Type: &cordiumv1.Workspace_Spec_Image_Dockerfile_Url{
							Url: "http://example.com/Dockerfile",
						},
					},
				},
			},
		},
		{
			Image: &cordiumv1.Workspace_Spec_Image{
				Type: &cordiumv1.Workspace_Spec_Image_Repository_{
					Repository: &cordiumv1.Workspace_Spec_Image_Repository{},
				},
			},
		},
		{
			Limit: &cordiumv1.Workspace_Spec_Limit{
				Cpu: &cordiumv1.Workspace_Spec_Limit_CPU{
					Millicores: maxLimitMillicores + 1,
				},
			},
		},
		{
			Config: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"agent": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"systemPromptAppend": structpb.NewStringValue(strings.Repeat("a", maxConfigBytes)),
						},
					}),
				},
			},
		},
	}

	for _, cfg := range invalidCfgs {
		err := ValidateConfig(cfg)
		assert.True(t, grpcerr.IsInvalidArg(err), "%+v: %+v", cfg, err)
	}
}
