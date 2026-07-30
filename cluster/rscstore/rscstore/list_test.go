// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
)

func TestIsSimpleFTSQuery(t *testing.T) {
	for _, query := range []string{
		"alice",
		"Alice Smith",
		"alice@example.com",
		"service-name",
		"service_name",
		"core:v1",
		"10.0.0.1",
	} {
		assert.True(t, isSimpleFTSQuery(query), query)
	}

	for _, query := range []string{
		"",
		"alice+smith",
		"alice/smith",
		"alice%",
		"alice_%",
		"こんにちは",
		strings.Repeat("a", 101),
	} {
		assert.False(t, isSimpleFTSQuery(query), query)
	}
}

func TestGetRSCStrIndexesExpectedResourceFields(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	data, err := json.Marshal(map[string]any{
		"apiVersion": "core/v1",
		"kind":       "User",
		"metadata": map[string]any{
			"name": "Alice",
			"tags": []any{"Engineering", "Production"},
		},
		"spec": map[string]any{
			"email":      "Alice@Example.COM",
			"isDisabled": false,
		},
		"status": map[string]any{
			"type": "HUMAN",
		},
		"ignoredTopLevel": "must-not-be-indexed",
	})
	assert.Nil(t, err, "%+v", err)

	ret, err := env.srv.getRSCStr(data)
	assert.Nil(t, err, "%+v", err)
	assert.Contains(t, ret, "alice")
	assert.Contains(t, ret, "engineering")
	assert.Contains(t, ret, "production")
	assert.Contains(t, ret, "alice@example.com")
	assert.Contains(t, ret, "human")
	assert.Contains(t, ret, "false")
	assert.NotContains(t, ret, "must-not-be-indexed")
}

func TestSearchTreatsLikeWildcardsAsLiteralCharacters(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	for idx := range 3 {
		insertRscStoreObject(t, env, &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata:   newRscStoreMetadata("wildcard-user-"+string(rune('a'+idx)), time.Now().Add(time.Duration(idx)*time.Second)),
			Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
			Status:     &corev1.User_Status{},
		})
	}

	resp, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			Query:        "%",
			ItemsPerPage: 1000,
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Empty(t, resp.Items)
	assert.Zero(t, resp.ListResponseMeta.TotalCount)
}

func TestSearchDoesNotReturnStaleFTSMatches(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	user := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("search-user", time.Now()),
		Spec: &corev1.User_Spec{
			Type:  corev1.User_Spec_HUMAN,
			Email: "old-search-value@example.com",
		},
		Status: &corev1.User_Status{},
	}
	insertRscStoreObject(t, env, user)

	err := env.srv.recreateFTSIndex(env.ctx)
	assert.Nil(t, err, "%+v", err)

	user.Metadata.ResourceVersion = vutils.UUIDv7()
	user.Spec.Email = "new-search-value@example.com"
	insertRscStoreObject(t, env, user)

	resp, err := (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			Query:        "old-search-value",
			ItemsPerPage: 1000,
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Empty(t, resp.Items)
	assert.Zero(t, resp.ListResponseMeta.TotalCount)

	resp, err = (&srvCore{s: env.srv}).ListUser(env.ctx, &vcorev1.ListUserOptions{
		Common: &vmetav1.CommonListOptions{
			Query:        "new-search-value",
			ItemsPerPage: 1000,
		},
	})
	assert.Nil(t, err, "%+v", err)
	assert.Len(t, resp.Items, 1)
}

func TestToResourceListPreservesItemsAndPagination(t *testing.T) {
	env := newRscStoreTestEnv(t)
	if env == nil {
		return
	}

	user1 := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("list-user-one", time.Now()),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_HUMAN},
		Status:     &corev1.User_Status{},
	}
	user2 := &corev1.User{
		ApiVersion: ucorev1.APIVersion,
		Kind:       ucorev1.KindUser,
		Metadata:   newRscStoreMetadata("list-user-two", time.Now().Add(time.Second)),
		Spec:       &corev1.User_Spec{Type: corev1.User_Spec_WORKLOAD},
		Status:     &corev1.User_Status{},
	}

	msg, err := env.srv.toResourceList([]umetav1.ResourceObjectI{user1, user2},
		&metav1.ListResponseMeta{
			Page:         2,
			ItemsPerPage: 10,
			TotalCount:   25,
			HasMore:      true,
		},
		ucorev1.API,
		ucorev1.Version,
		ucorev1.KindUser,
	)
	assert.Nil(t, err, "%+v", err)

	resp, ok := msg.(*corev1.UserList)
	assert.True(t, ok)
	if !ok {
		return
	}

	assert.Len(t, resp.Items, 2)
	assert.True(t, pbutils.IsEqual(user1, resp.Items[0]))
	assert.True(t, pbutils.IsEqual(user2, resp.Items[1]))
	assert.Equal(t, uint32(2), resp.ListResponseMeta.Page)
	assert.Equal(t, uint32(10), resp.ListResponseMeta.ItemsPerPage)
	assert.Equal(t, uint32(25), resp.ListResponseMeta.TotalCount)
	assert.True(t, resp.ListResponseMeta.HasMore)
}
