// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package suite

import (
	"context"
	"fmt"
	"slices"
	"testing"

	eeharness "github.com/octelium/octelium-ee/cluster/e2e/harness"
	"github.com/octelium/octelium/apis/main/cordiumv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/e2e/harness"
	"github.com/octelium/octelium/pkg/grpcerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAgent(t *testing.T, ch *harness.H) {
	h := eeharness.Wrap(ch)

	ctx := t.Context()

	actor := h.NewActorWithAuthorization(t, &corev1.User_Spec_Authorization{
		InlinePolicies: eeharness.APIPolicy("agent-api",
			eeharness.AgentService, eeharness.CordiumMainService),
	})

	agentC := h.AgentC(actor.Conn)
	cordiumC := h.CordiumC(actor.Conn)

	if !h.IsCordiumInstalled(t) {
		t.Run("ItIsUnavailableWithoutCordium", func(t *testing.T) {
			_, err := agentC.GetAgent(ctx, &enterprisev1.GetAgentRequest{})
			require.NotNil(t, err)
			assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)

			_, err = agentC.InitializeAgent(ctx, &enterprisev1.InitializeAgentRequest{})
			require.NotNil(t, err)
			assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)

			_, err = agentC.CreateAgentWorkspace(ctx, &enterprisev1.CreateAgentWorkspaceRequest{})
			require.NotNil(t, err)
			assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
		})

		return
	}

	var agent *enterprisev1.Agent

	t.Cleanup(func() {
		if agent == nil || agent.SpaceRef == nil {
			return
		}

		cordiumC.DeleteSpace(context.Background(), &metav1.DeleteOptions{Uid: agent.SpaceRef.Uid})
	})

	t.Run("NotInitializedForANewUser", func(t *testing.T) {
		res, err := agentC.GetAgent(ctx, &enterprisev1.GetAgentRequest{})
		require.Nil(t, err)
		assert.Equal(t, enterprisev1.Agent_NOT_INITIALIZED, res.State)
		assert.Empty(t, res.Workspaces)

		_, err = agentC.CreateAgentWorkspace(ctx, &enterprisev1.CreateAgentWorkspaceRequest{})
		require.NotNil(t, err)
		assert.True(t, grpcerr.IsFailedPrecondition(err), "%+v", err)
	})

	t.Run("InitializeAgentProvisionsTheEnvironment", func(t *testing.T) {
		res, err := agentC.InitializeAgent(ctx, &enterprisev1.InitializeAgentRequest{})
		require.Nil(t, err)
		agent = res

		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		assert.Equal(t, fmt.Sprintf("octelium.%s", actor.User.Metadata.Name), res.SpaceRef.Name)
		require.Len(t, res.Workspaces, 1)
		assert.Equal(t, enterprisev1.Agent_Workspace_PRIMARY, res.Workspaces[0].Type)
		assert.Equal(t, cordiumv1.Workspace_Status_STOPPED, res.Workspaces[0].Workspace.Status.State)
	})

	require.NotNil(t, agent)
	primary := agent.Workspaces[0].Workspace

	t.Run("InitializeAgentIsIdempotent", func(t *testing.T) {
		res, err := agentC.InitializeAgent(ctx, &enterprisev1.InitializeAgentRequest{})
		require.Nil(t, err)
		assert.Equal(t, enterprisev1.Agent_READY, res.State)
		require.Len(t, res.Workspaces, 1)
		assert.Equal(t, primary.Metadata.Uid, res.Workspaces[0].Workspace.Metadata.Uid)
	})

	t.Run("TheEnvironmentIsVisibleFromCordium", func(t *testing.T) {
		spcList, err := cordiumC.ListSpace(ctx, &cordiumv1.ListSpaceOptions{})
		require.Nil(t, err)
		assert.True(t, slices.ContainsFunc(spcList.Items, func(itm *cordiumv1.Space) bool {
			return itm.Metadata.Uid == agent.SpaceRef.Uid
		}))

		spc, err := cordiumC.GetSpace(ctx, &metav1.GetOptions{Uid: agent.SpaceRef.Uid})
		require.Nil(t, err)
		assert.True(t, spc.Metadata.IsSystem)
		assert.Equal(t, cordiumv1.Space_Status_USER, spc.Status.Type)

		ws, err := cordiumC.GetWorkspace(ctx, &metav1.GetOptions{Uid: primary.Metadata.Uid})
		require.Nil(t, err)
		assert.Equal(t, agent.SpaceRef.Uid, ws.Status.SpaceRef.Uid)
		assert.Equal(t, agent.TemplateRef.Uid, ws.Status.TemplateRef.Uid)

		tmpl, err := cordiumC.GetTemplate(ctx, &metav1.GetOptions{Uid: agent.TemplateRef.Uid})
		require.Nil(t, err)
		require.NotNil(t, tmpl.Spec.Runtime)
		assert.Len(t, tmpl.Spec.Runtime.Tasks, 2)
	})

	t.Run("CreateAgentWorkspaceAddsAWorkspace", func(t *testing.T) {
		ws, err := agentC.CreateAgentWorkspace(ctx, &enterprisev1.CreateAgentWorkspaceRequest{
			DisplayName: "Fresh",
			IsEphemeral: true,
		})
		require.Nil(t, err)
		assert.Equal(t, enterprisev1.Agent_Workspace_ADDITIONAL, ws.Type)
		assert.True(t, ws.Workspace.Spec.IsEphemeral)

		res, err := agentC.GetAgent(ctx, &enterprisev1.GetAgentRequest{})
		require.Nil(t, err)
		require.Len(t, res.Workspaces, 2)
		assert.Equal(t, primary.Metadata.Uid, res.Workspaces[0].Workspace.Metadata.Uid)
		assert.Equal(t, ws.Workspace.Metadata.Uid, res.Workspaces[1].Workspace.Metadata.Uid)
	})

	t.Run("DeletingTheSpaceViaCordiumResetsTheAgent", func(t *testing.T) {
		_, err := cordiumC.DeleteSpace(ctx, &metav1.DeleteOptions{Uid: agent.SpaceRef.Uid})
		require.Nil(t, err)

		h.Eventually(t, "the agent to be reset", eeharness.PropagationBudget,
			func(ctx context.Context) error {
				res, err := agentC.GetAgent(ctx, &enterprisev1.GetAgentRequest{})
				if err != nil {
					return err
				}
				if res.State != enterprisev1.Agent_NOT_INITIALIZED {
					return errors.Errorf("the agent state is %s", res.State)
				}
				return nil
			})

		agent = nil
	})
}
