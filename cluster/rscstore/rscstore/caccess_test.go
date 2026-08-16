// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/cluster/caccessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/stretchr/testify/assert"
)

func insertSubjectUser(t *testing.T, env *rscStoreTestEnv, usr *corev1.User) {
	t.Helper()

	usr.ApiVersion = ucorev1.APIVersion
	usr.Kind = ucorev1.KindUser
	if usr.Status == nil {
		usr.Status = &corev1.User_Status{}
	}

	insertRscStoreObject(t, env, usr)
}

func subjectUserNames(lst *corev1.UserList) []string {
	ret := []string{}
	for _, itm := range lst.Items {
		ret = append(ret, itm.Metadata.Name)
	}

	return ret
}

func TestListSubjectUser(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	now := time.Now().UTC()

	insertSubjectUser(t, env, &corev1.User{
		Metadata: newRscStoreMetadata("alice", now),
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: "alice@example.com",
		},
	})

	{
		metadata := newRscStoreMetadata("bob", now)
		metadata.DisplayName = "Bob Alicorn"
		insertSubjectUser(t, env, &corev1.User{
			Metadata: metadata,
			Spec: &corev1.User_Spec{
				Type:  corev1.User_Spec_HUMAN,
				Email: "bob@example.com",
			},
		})
	}

	insertSubjectUser(t, env, &corev1.User{
		Metadata: newRscStoreMetadata("carol", now),
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: "carol@example.com",
		},
	})

	insertSubjectUser(t, env, &corev1.User{
		Metadata: newRscStoreMetadata("workload-one", now),
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_WORKLOAD,
			Email: "alice-workload@example.com",
		},
	})

	insertSubjectUser(t, env, &corev1.User{
		Metadata: newRscStoreMetadata("disabled-alice", now),
		Spec: &corev1.User_Spec{
			Type:       corev1.User_Spec_HUMAN,
			Email:      "disabled-alice@example.com",
			IsDisabled: true,
		},
	})

	{
		metadata := newRscStoreMetadata("hidden-alice", now)
		metadata.IsSystemHidden = true
		insertSubjectUser(t, env, &corev1.User{
			Metadata: metadata,
			Spec: &corev1.User_Spec{
				Type:  corev1.User_Spec_HUMAN,
				Email: "hidden-alice@example.com",
			},
		})
	}

	{
		metadata := newRscStoreMetadata("dave", now)
		metadata.Annotations = map[string]string{
			"secret": "topsecretvalue",
		}
		insertSubjectUser(t, env, &corev1.User{
			Metadata: metadata,
			Spec: &corev1.User_Spec{
				Type:   corev1.User_Spec_HUMAN,
				Email:  "dave@example.com",
				Groups: []string{"topsecretgroup"},
			},
		})
	}

	srv := &srvClusterAccess{s: env.srv}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"alice", "bob", "carol", "dave"}, subjectUserNames(resp))
		assert.Equal(t, uint32(4), resp.ListResponseMeta.TotalCount)
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: "alic"})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"alice", "bob"}, subjectUserNames(resp))
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: "ALICE"})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"alice"}, subjectUserNames(resp))
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: "carol@example.com"})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"carol"}, subjectUserNames(resp))
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: "  bob  "})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"bob"}, subjectUserNames(resp))
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: "nosuchuser"})
		assert.Nil(t, err, "%+v", err)
		assert.Empty(t, resp.Items)
		assert.Equal(t, uint32(0), resp.ListResponseMeta.TotalCount)
	}

	for _, query := range []string{"topsecretvalue", "topsecretgroup", "HUMAN"} {
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: query})
		assert.Nil(t, err, "%+v", err)
		assert.Empty(t, resp.Items, "query %s must not match non searchable fields", query)
	}

	for _, query := range []string{"%", "_", `\`, "a%e", "a_ice"} {
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{Query: query})
		assert.Nil(t, err, "%+v", err)
		assert.Empty(t, resp.Items, "query %s must be matched literally", query)
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{ItemsPerPage: 2})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"alice", "bob"}, subjectUserNames(resp))
		assert.Equal(t, uint32(4), resp.ListResponseMeta.TotalCount)
		assert.True(t, resp.ListResponseMeta.HasMore)
	}

	{
		resp, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{ItemsPerPage: 2, Page: 1})
		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, []string{"carol", "dave"}, subjectUserNames(resp))
		assert.False(t, resp.ListResponseMeta.HasMore)
	}

	{
		_, err := srv.ListSubjectUser(env.ctx, &caccessv1.ListSubjectUserRequest{
			Query: strings.Repeat("a", 101),
		})
		assert.NotNil(t, err)
	}
}
