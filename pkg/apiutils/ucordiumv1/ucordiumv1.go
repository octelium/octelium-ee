// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ucordiumv1

import (
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/pkg/errors"
)

const (
	KindWorkspace         = "Workspace"
	KindWorkspaceSnapshot = "WorkspaceSnapshot"
	KindVolume            = "Volume"
	KindSecret            = "Secret"
	KindTemplate          = "Template"
	KindSpace             = "Space"
	KindMembership        = "Membership"
	KindGitProvider       = "GitProvider"
	KindUserSecret        = "UserSecret"
	KindUserConfig        = "UserConfig"
	KindClusterConfig     = "ClusterConfig"
)

type ResourceObjectRefG interface {
	*cordiumv1.Workspace | *cordiumv1.WorkspaceSnapshot | *cordiumv1.Volume |
		*cordiumv1.Secret | *cordiumv1.Template | *cordiumv1.Space | *cordiumv1.Membership |
		*cordiumv1.GitProvider | *cordiumv1.UserSecret | *cordiumv1.UserConfig | *cordiumv1.ClusterConfig
}

const API = "cordium"
const Version = "v1"
const APIVersion = "cordium/v1"

func NewObjectList(kind string) (umetav1.ObjectI, error) {

	switch kind {
	case KindWorkspace:
		return &cordiumv1.WorkspaceList{}, nil
	case KindWorkspaceSnapshot:
		return &cordiumv1.WorkspaceSnapshotList{}, nil
	case KindVolume:
		return &cordiumv1.VolumeList{}, nil
	case KindSecret:
		return &cordiumv1.SecretList{}, nil
	case KindTemplate:
		return &cordiumv1.TemplateList{}, nil
	case KindSpace:
		return &cordiumv1.SpaceList{}, nil
	case KindMembership:
		return &cordiumv1.MembershipList{}, nil
	case KindGitProvider:
		return &cordiumv1.GitProviderList{}, nil
	case KindUserSecret:
		return &cordiumv1.UserSecretList{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

func NewObject(kind string) (umetav1.ResourceObjectI, error) {

	switch kind {
	case KindWorkspace:
		return &cordiumv1.Workspace{}, nil
	case KindWorkspaceSnapshot:
		return &cordiumv1.WorkspaceSnapshot{}, nil
	case KindVolume:
		return &cordiumv1.Volume{}, nil
	case KindSecret:
		return &cordiumv1.Secret{}, nil
	case KindTemplate:
		return &cordiumv1.Template{}, nil
	case KindSpace:
		return &cordiumv1.Space{}, nil
	case KindMembership:
		return &cordiumv1.Membership{}, nil
	case KindGitProvider:
		return &cordiumv1.GitProvider{}, nil
	case KindUserSecret:
		return &cordiumv1.UserSecret{}, nil
	case KindUserConfig:
		return &cordiumv1.UserConfig{}, nil
	case KindClusterConfig:
		return &cordiumv1.ClusterConfig{}, nil
	default:
		return nil, errors.Errorf("Invalid kind: %s", kind)
	}
}

type Workspace struct {
	*cordiumv1.Workspace
}

func ToWorkspace(a *cordiumv1.Workspace) *Workspace {
	return &Workspace{
		Workspace: a,
	}
}

func (w *Workspace) IsStopped() bool {
	return w.Status != nil && w.Status.State == cordiumv1.Workspace_Status_STOPPED
}

func (w *Workspace) IsRunning() bool {
	return w.Status != nil && w.Status.State == cordiumv1.Workspace_Status_RUNNING
}
