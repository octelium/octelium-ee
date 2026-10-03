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
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/rsc/rlockv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/urscsrv"
	"github.com/octelium/octelium/cluster/common/userctx"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	spaceBaseName    = "octelium"
	templateBaseName = "default"

	systemLabelAgent        = "octelium-agent"
	workspaceTypePrimary    = "primary"
	workspaceTypeAdditional = "additional"

	versionInfoKeyCordium = "cordium"

	maxDisplayNameLen    = 120
	maxWorkspacesPerUser = 1000

	lockTTLSeconds  = 60
	lockWaitSeconds = 30
)

type Server struct {
	octeliumC octeliumc.ClientInterface
	enterprisev1.UnimplementedAgentServiceServer
}

func NewServer(octeliumC octeliumc.ClientInterface) *Server {
	return &Server{
		octeliumC: octeliumC,
	}
}

func getSpaceName(usr *corev1.User) string {
	return fmt.Sprintf("%s.%s", spaceBaseName, usr.Metadata.Name)
}

func getTemplateName(usr *corev1.User) string {
	return fmt.Sprintf("%s.%s", templateBaseName, getSpaceName(usr))
}

func (s *Server) GetAgent(ctx context.Context, req *enterprisev1.GetAgentRequest) (*enterprisev1.Agent, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := s.getConfig(ctx)
	if err != nil {
		return nil, err
	}

	spc, err := s.getSpace(ctx, i.User)
	if err != nil {
		return nil, err
	}

	if spc == nil {
		return &enterprisev1.Agent{
			State: enterprisev1.Agent_NOT_INITIALIZED,
		}, nil
	}

	return s.getAgent(ctx, i.User, spc, cfg)
}

func (s *Server) InitializeAgent(ctx context.Context, req *enterprisev1.InitializeAgentRequest) (*enterprisev1.Agent, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := s.getConfig(ctx)
	if err != nil {
		return nil, err
	}

	tmplSpec, err := getTemplateSpec(cfg)
	if err != nil {
		return nil, err
	}

	unlock, err := s.lock(ctx, i.User)
	if err != nil {
		return nil, err
	}
	defer unlock()

	spc, err := s.getSpace(ctx, i.User)
	if err != nil {
		return nil, err
	}

	if spc == nil {
		spc, err = s.createSpace(ctx, i.User)
		if err != nil {
			return nil, err
		}
	}

	tmpl, err := s.setTemplate(ctx, i.User, spc, tmplSpec)
	if err != nil {
		return nil, err
	}

	wsList, err := s.listWorkspaces(ctx, i.User, spc)
	if err != nil {
		return nil, err
	}

	if !slices.ContainsFunc(wsList, isPrimaryWorkspace) {
		if _, err := s.createWorkspace(ctx, i.User, spc, tmpl, &createWorkspaceOpts{
			typ:         workspaceTypePrimary,
			displayName: "Octelium Agent",
		}); err != nil {
			return nil, err
		}
	}

	return s.getAgent(ctx, i.User, spc, cfg)
}

func (s *Server) CreateAgentWorkspace(ctx context.Context,
	req *enterprisev1.CreateAgentWorkspaceRequest) (*enterprisev1.Agent_Workspace, error) {
	if req == nil {
		return nil, grpcutils.InvalidArg("Nil request")
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if !utf8.ValidString(displayName) || len(displayName) > maxDisplayNameLen {
		return nil, grpcutils.InvalidArg("Invalid displayName")
	}

	i, err := userctx.GetUserCtx(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := s.getConfig(ctx); err != nil {
		return nil, err
	}

	spc, err := s.getSpace(ctx, i.User)
	if err != nil {
		return nil, err
	}
	if spc == nil {
		return nil, status.Error(codes.FailedPrecondition, "The agent is not initialized yet")
	}

	tmpl, err := s.octeliumC.CordiumC().GetTemplate(ctx, &rmetav1.GetOptions{
		Name: getTemplateName(i.User),
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, status.Error(codes.FailedPrecondition, "The agent Template does not exist. Initialize the agent first")
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	if displayName == "" {
		displayName = "Octelium Agent"
	}

	ws, err := s.createWorkspace(ctx, i.User, spc, tmpl, &createWorkspaceOpts{
		typ:         workspaceTypeAdditional,
		displayName: displayName,
		isEphemeral: req.IsEphemeral,
	})
	if err != nil {
		return nil, err
	}

	return toAgentWorkspace(ws), nil
}

func (s *Server) getConfig(ctx context.Context) (*enterprisev1.ClusterConfig_Spec_Agent, error) {
	cc, err := s.octeliumC.EnterpriseV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	cfg := cc.GetSpec().GetAgent()
	if cfg.GetIsDisabled() {
		return nil, status.Error(codes.FailedPrecondition, "The agent is disabled in this Cluster")
	}

	rgn, err := s.octeliumC.CoreC().GetRegion(ctx, &rmetav1.GetOptions{
		Name: "default",
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if _, ok := rgn.GetStatus().GetVersionInfoMap()[versionInfoKeyCordium]; !ok {
		return nil, status.Error(codes.FailedPrecondition, "Cordium is not installed in this Cluster")
	}

	return cfg, nil
}

func (s *Server) lock(ctx context.Context, usr *corev1.User) (func(), error) {
	key := []byte(fmt.Sprintf("apiserver.agent.%s", usr.Metadata.Uid))

	res, err := s.octeliumC.LockC().Lock(ctx, &rlockv1.LockRequest{
		Key: key,
		Ttl: &metav1.Duration{
			Type: &metav1.Duration_Seconds{
				Seconds: lockTTLSeconds,
			},
		},
		Wait: &metav1.Duration{
			Type: &metav1.Duration_Seconds{
				Seconds: lockWaitSeconds,
			},
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	if !res.Acquired {
		return nil, status.Error(codes.Aborted, "The agent is currently being initialized. Try again later")
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, err := s.octeliumC.LockC().Unlock(ctx, &rlockv1.UnlockRequest{
			Key:     key,
			LeaseID: res.LeaseID,
		}); err != nil {
			zap.L().Warn("Could not release the agent lock", zap.Error(err))
		}
	}, nil
}

func (s *Server) getSpace(ctx context.Context, usr *corev1.User) (*cordiumv1.Space, error) {
	spc, err := s.octeliumC.CordiumC().GetSpace(ctx, &rmetav1.GetOptions{
		Name: getSpaceName(usr),
	})
	if err != nil {
		if grpcerr.IsNotFound(err) {
			return nil, nil
		}
		return nil, grpcutils.InternalWithErr(err)
	}

	if spc.Metadata.SystemLabels[systemLabelAgent] != "true" ||
		spc.GetStatus().GetUserRef().GetUid() != usr.Metadata.Uid {
		return nil, grpcutils.AlreadyExists(
			"The Space %s already exists and it is not managed by the Cluster", spc.Metadata.Name)
	}

	return spc, nil
}

func (s *Server) createSpace(ctx context.Context, usr *corev1.User) (*cordiumv1.Space, error) {
	spc, err := s.octeliumC.CordiumC().CreateSpace(ctx, &cordiumv1.Space{
		Metadata: &metav1.Metadata{
			Name:        getSpaceName(usr),
			DisplayName: "Octelium",
			Description: "The personal Space of the Octelium console AI agent",
			IsSystem:    true,
			SystemLabels: map[string]string{
				"type":           "user",
				systemLabelAgent: "true",
			},
		},
		Spec: &cordiumv1.Space_Spec{},
		Status: &cordiumv1.Space_Status{
			UserRef: umetav1.GetObjectReference(usr),
			Type:    cordiumv1.Space_Status_USER,
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	mem := &cordiumv1.Membership{
		Metadata: &metav1.Metadata{
			Name: fmt.Sprintf("%s.%s", usr.Metadata.Name, spc.Metadata.Name),
		},
		Spec: &cordiumv1.Membership_Spec{
			Role: cordiumv1.Membership_Spec_OWNER,
		},
		Status: &cordiumv1.Membership_Status{
			UserRef:  umetav1.GetObjectReference(usr),
			SpaceRef: umetav1.GetObjectReference(spc),
			UserInfo: &cordiumv1.Membership_Status_UserInfo{
				DisplayName: usr.Metadata.DisplayName,
				PicURL:      usr.Metadata.PicURL,
			},
		},
	}

	if _, err := s.octeliumC.CordiumC().CreateMembership(ctx, mem); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return spc, nil
}

func (s *Server) setTemplate(ctx context.Context,
	usr *corev1.User, spc *cordiumv1.Space, spec *cordiumv1.Template_Spec) (*cordiumv1.Template, error) {
	tmpl, err := s.octeliumC.CordiumC().GetTemplate(ctx, &rmetav1.GetOptions{
		Name: getTemplateName(usr),
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}

		tmpl, err = s.octeliumC.CordiumC().CreateTemplate(ctx, &cordiumv1.Template{
			Metadata: &metav1.Metadata{
				Name:        getTemplateName(usr),
				DisplayName: "Octelium Agent",
				Description: "The Template of the Octelium console AI agent Workspaces",
				IsSystem:    true,
				SystemLabels: map[string]string{
					systemLabelAgent: "true",
				},
			},
			Spec: spec,
			Status: &cordiumv1.Template_Status{
				SpaceRef:  umetav1.GetObjectReference(spc),
				UserRef:   umetav1.GetObjectReference(usr),
				BuildInfo: &cordiumv1.Template_Status_BuildInfo{},
			},
		})
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		return tmpl, nil
	}

	if pbutils.IsEqual(tmpl.Spec, spec) {
		return tmpl, nil
	}

	tmpl.Spec = spec
	tmpl, err = s.octeliumC.CordiumC().UpdateTemplate(ctx, tmpl)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return tmpl, nil
}

type createWorkspaceOpts struct {
	typ         string
	displayName string
	isEphemeral bool
}

func (s *Server) createWorkspace(ctx context.Context,
	usr *corev1.User, spc *cordiumv1.Space, tmpl *cordiumv1.Template,
	o *createWorkspaceOpts) (*cordiumv1.Workspace, error) {

	cc, err := s.octeliumC.CordiumV1Utils().GetClusterConfig(ctx)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	listOpts := urscsrv.FilterByUser(usr)
	listOpts.Paginate = true
	listOpts.ItemsPerPage = 1

	wsList, err := s.octeliumC.CordiumC().ListWorkspace(ctx, listOpts)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if wsList.GetListResponseMeta().GetTotalCount() >= uint32(getMaxWorkspacesPerUser(cc)) {
		return nil, grpcutils.InvalidArg("Number of Workspaces per User has been exceeded")
	}

	name, err := s.genWorkspaceName(ctx)
	if err != nil {
		return nil, err
	}

	ws, err := s.octeliumC.CordiumC().CreateWorkspace(ctx, &cordiumv1.Workspace{
		Metadata: &metav1.Metadata{
			Name:        name,
			DisplayName: o.displayName,
			SystemLabels: map[string]string{
				systemLabelAgent: o.typ,
			},
		},
		Spec: &cordiumv1.Workspace_Spec{
			Applications: []*cordiumv1.Workspace_Spec_Application{
				{
					Name:        applicationName,
					DisplayName: "Octelium Agent",
					Port:        applicationPort,
					IsDefault:   true,
				},
			},
			IsEphemeral: o.isEphemeral,
		},
		Status: &cordiumv1.Workspace_Status{
			UserRef:     umetav1.GetObjectReference(usr),
			State:       cordiumv1.Workspace_Status_STOPPED,
			TemplateRef: umetav1.GetObjectReference(tmpl),
			SpaceRef:    umetav1.GetObjectReference(spc),
			SpaceType:   spc.Status.Type,
			Limit:       &cordiumv1.Workspace_Spec_Limit{},
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ws, nil
}

func getMaxWorkspacesPerUser(cc *cordiumv1.ClusterConfig) int {
	limit := cc.GetSpec().GetWorkspace().GetLimit()
	if limit.GetMaxPerUser() != 0 && limit.GetMaxPerUser() < 1000000 {
		return int(limit.GetMaxPerUser())
	}

	return maxWorkspacesPerUser
}

func (s *Server) genWorkspaceName(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for i := 0; i < 10000; i++ {
		var name string

		switch {
		case i < 50:
			name = utilrand.GetRandomStringCanonical(3)
		case i < 500:
			name = utilrand.GetRandomStringCanonical(4)
		case i < 1000:
			name = utilrand.GetRandomStringCanonical(5)
		default:
			name = utilrand.GetRandomStringCanonical(6)
		}

		if _, err := s.octeliumC.CordiumC().GetWorkspace(ctx, &rmetav1.GetOptions{
			Name: name,
		}); err != nil {
			if grpcerr.IsNotFound(err) {
				return name, nil
			}

			return "", grpcutils.InternalWithErr(err)
		}
	}

	return "", grpcutils.Internal("Could not generate Workspace name")
}

func (s *Server) listWorkspaces(ctx context.Context,
	usr *corev1.User, spc *cordiumv1.Space) ([]*cordiumv1.Workspace, error) {
	itemList, err := s.octeliumC.CordiumC().ListWorkspace(ctx, &rmetav1.ListOptions{
		Filters: []*rmetav1.ListOptions_Filter{
			urscsrv.FilterStatusUserUID(usr.Metadata.Uid),
			urscsrv.FilterFieldEQValStr("status.spaceRef.uid", spc.Metadata.Uid),
		},
	})
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var ret []*cordiumv1.Workspace
	for _, itm := range itemList.Items {
		if itm.GetStatus().GetIsBuild() || itm.Metadata.IsSystemHidden || itm.Metadata.IsUserHidden {
			continue
		}
		ret = append(ret, itm)
	}

	return ret, nil
}

func (s *Server) getAgent(ctx context.Context,
	usr *corev1.User, spc *cordiumv1.Space, cfg *enterprisev1.ClusterConfig_Spec_Agent) (*enterprisev1.Agent, error) {
	ret := &enterprisev1.Agent{
		State:    enterprisev1.Agent_READY,
		SpaceRef: umetav1.GetObjectReference(spc),
	}

	tmpl, err := s.octeliumC.CordiumC().GetTemplate(ctx, &rmetav1.GetOptions{
		Name: getTemplateName(usr),
	})
	if err != nil {
		if !grpcerr.IsNotFound(err) {
			return nil, grpcutils.InternalWithErr(err)
		}
		ret.State = enterprisev1.Agent_OUTDATED
	} else {
		ret.TemplateRef = umetav1.GetObjectReference(tmpl)

		spec, err := getTemplateSpec(cfg)
		if err != nil {
			return nil, err
		}
		if !pbutils.IsEqual(tmpl.Spec, spec) {
			ret.State = enterprisev1.Agent_OUTDATED
		}
	}

	wsList, err := s.listWorkspaces(ctx, usr, spc)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(wsList, func(a, b *cordiumv1.Workspace) int {
		switch {
		case isPrimaryWorkspace(a) && !isPrimaryWorkspace(b):
			return -1
		case !isPrimaryWorkspace(a) && isPrimaryWorkspace(b):
			return 1
		default:
			return b.Metadata.CreatedAt.AsTime().Compare(a.Metadata.CreatedAt.AsTime())
		}
	})

	if !slices.ContainsFunc(wsList, isPrimaryWorkspace) {
		ret.State = enterprisev1.Agent_OUTDATED
	}

	for _, ws := range wsList {
		ret.Workspaces = append(ret.Workspaces, toAgentWorkspace(ws))
	}

	return ret, nil
}

func isPrimaryWorkspace(ws *cordiumv1.Workspace) bool {
	return ws.Metadata.SystemLabels[systemLabelAgent] == workspaceTypePrimary
}

func toAgentWorkspace(ws *cordiumv1.Workspace) *enterprisev1.Agent_Workspace {
	ret := &enterprisev1.Agent_Workspace{
		Type:      enterprisev1.Agent_Workspace_ADDITIONAL,
		Workspace: ws,
	}

	if isPrimaryWorkspace(ws) {
		ret.Type = enterprisev1.Agent_Workspace_PRIMARY
	}

	if hostname := ws.GetStatus().GetHostname(); hostname != "" {
		ret.Url = fmt.Sprintf("https://%s", hostname)
	}

	return ret
}
