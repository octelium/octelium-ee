// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSessionCountry(t *testing.T) {
	env := newRscStoreTestEnv(t)
	require.NotNil(t, env)

	srvCore := &srvCore{s: env.srv}
	now := time.Now().UTC()

	newSession := func(name string, createdAt time.Time, geoip *corev1.GeoIP) *corev1.Session {
		return &corev1.Session{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindSession,
			Metadata:   newRscStoreMetadata(name, createdAt),
			Spec:       &corev1.Session_Spec{State: corev1.Session_Spec_ACTIVE},
			Status: &corev1.Session_Status{
				Type: corev1.Session_Status_CLIENTLESS,
				Authentication: &corev1.Session_Status_Authentication{
					Info: &corev1.Session_Status_Authentication_Info{
						Type:  corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER,
						Geoip: geoip,
					},
				},
			},
		}
	}

	geoip := func(code string) *corev1.GeoIP {
		return &corev1.GeoIP{Country: &corev1.GeoIP_Country{Code: code}}
	}

	insertRscStoreObject(t, env, newSession("sess-us-1", now.Add(-2*time.Hour), geoip("US")))
	insertRscStoreObject(t, env, newSession("sess-us-2", now.Add(-time.Minute), geoip("US")))
	insertRscStoreObject(t, env, newSession("sess-de", now.Add(-time.Minute), geoip("de")))
	insertRscStoreObject(t, env, newSession("sess-no-geo", now.Add(-time.Minute), nil))
	insertRscStoreObject(t, env, newSession("sess-no-country", now.Add(-time.Minute), &corev1.GeoIP{Ip: "1.2.3.4"}))

	{
		resp, err := env.srv.getSummaryCoreSession(env.ctx, &vcorev1.GetSessionSummaryRequest{})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(5), resp.TotalNumber)
		assert.Equal(t, map[string]uint64{"US": 2, "DE": 1}, resp.TotalByCountry)
	}

	{
		resp, err := env.srv.getSummaryCoreSession(env.ctx, &vcorev1.GetSessionSummaryRequest{
			Common: &vmetav1.CommonSummaryOptions{
				From:        pbutils.Timestamp(now.Add(-time.Hour)),
				To:          pbutils.Timestamp(now.Add(time.Hour)),
				CompareFrom: pbutils.Timestamp(now.Add(-3 * time.Hour)),
				CompareTo:   pbutils.Timestamp(now.Add(-time.Hour)),
			},
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(4), resp.TotalNumber)
		assert.Equal(t, map[string]uint64{"US": 1, "DE": 1}, resp.TotalByCountry)
		require.NotNil(t, resp.Previous)
		assert.Equal(t, map[string]uint64{"US": 1}, resp.Previous.TotalByCountry)
	}

	for _, tt := range []struct {
		code  string
		names []string
	}{
		{code: "US", names: []string{"sess-us-1", "sess-us-2"}},
		{code: "de", names: []string{"sess-de"}},
		{code: " De ", names: []string{"sess-de"}},
		{code: "FR"},
	} {
		resp, err := srvCore.ListSession(env.ctx, &vcorev1.ListSessionOptions{
			CountryCode: tt.code,
		})
		require.Nil(t, err, "%+v", err)

		var names []string
		for _, itm := range resp.Items {
			names = append(names, itm.Metadata.Name)
		}
		assert.ElementsMatch(t, tt.names, names, tt.code)
	}

	{
		resp, err := srvCore.ListSession(env.ctx, &vcorev1.ListSessionOptions{
			CountryCode: "US",
			Type:        corev1.Session_Status_CLIENT,
		})
		require.Nil(t, err, "%+v", err)
		assert.Len(t, resp.Items, 0)
	}

	for _, code := range []string{"USA", "U", "1A", "U'"} {
		_, err := srvCore.ListSession(env.ctx, &vcorev1.ListSessionOptions{
			CountryCode: code,
		})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), code)
	}
}
