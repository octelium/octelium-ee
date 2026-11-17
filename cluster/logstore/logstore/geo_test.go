// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/enterprisev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestGeoIP(countryCode, countryName, city string, lat, lng float64) *corev1.GeoIP {
	ret := &corev1.GeoIP{
		Country: &corev1.GeoIP_Country{
			Code: countryCode,
			Name: countryName,
		},
		Continent: &corev1.GeoIP_Continent{
			Code: "EU",
			Name: "Europe",
		},
		Coordinates: &corev1.GeoIP_Coordinates{
			Latitude:       lat,
			Longitude:      lng,
			AccuracyRadius: 20,
		},
	}

	if city != "" {
		ret.City = &corev1.GeoIP_City{Name: city}
		ret.Region = &corev1.GeoIP_Region{Code: "R1", Name: fmt.Sprintf("%s region", city)}
	}

	return ret
}

func getTestSessionLocation(t *testing.T, srv *Server, sessionUID string) sql.NullString {
	t.Helper()

	var ret sql.NullString
	err := srv.db.QueryRow(fmt.Sprintf(`SELECT %s FROM %s WHERE %s = ?`,
		colCountryCode, tableSessionLocations, colSessionUID), sessionUID).Scan(&ret)
	if err == sql.ErrNoRows {
		return sql.NullString{}
	}
	assert.Nil(t, err, "%+v", err)
	return ret
}

func getTestAccessLogCountry(t *testing.T, srv *Server, id string) sql.NullString {
	t.Helper()

	var ret sql.NullString
	err := srv.db.QueryRow(fmt.Sprintf(`SELECT %s FROM access_logs WHERE json_extract_string(rsc, '$.metadata.id') = ?`,
		colCountryCode), id).Scan(&ret)
	assert.Nil(t, err, "%+v", err)
	return ret
}

func TestGetCountryCode(t *testing.T) {
	for _, tt := range []struct {
		code    string
		want    string
		invalid bool
	}{
		{code: "US", want: "US"},
		{code: " de ", want: "DE"},
		{code: "gb", want: "GB"},
		{code: "USA", invalid: true},
		{code: "U", invalid: true},
		{code: "U1", invalid: true},
		{code: "é1", invalid: true},
		{code: "'X", invalid: true},
	} {
		ret, err := getCountryCode(tt.code)
		if tt.invalid {
			assert.Equal(t, codes.InvalidArgument, status.Code(err), tt.code)
			continue
		}

		assert.Nil(t, err, "%+v", err)
		assert.Equal(t, tt.want, ret)
	}
}

func TestGetGeoTopLimit(t *testing.T) {
	assert.Equal(t, defaultTopItems, getGeoTopLimit(0))
	assert.Equal(t, 1, getGeoTopLimit(1))
	assert.Equal(t, 250, getGeoTopLimit(250))
	assert.Equal(t, maxGeoTopItems, getGeoTopLimit(1000))
}

func TestAccessLogSessionLocation(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)

	base := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Second)

	sessUS := randomObjectReference()
	sessDE := randomObjectReference()
	sessUnknown := randomObjectReference()
	sessNoGeo := randomObjectReference()
	userRef := randomObjectReference()

	authLog := func(at time.Duration, sessionRef *metav1.ObjectReference, geoip *corev1.GeoIP,
		st enterprisev1.AuthenticationLog_Entry_Status) pendingLog {
		return pendingLog{
			table: logTableAuthentication,
			data: marshalLog(t, newAuthenticationLog(&authenticationLogOptions{
				CreatedAt:  base.Add(at),
				UserRef:    userRef,
				SessionRef: sessionRef,
				Type:       corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER,
				Status:     st,
				GeoIP:      geoip,
				FailureReason: func() enterprisev1.AuthenticationLog_Entry_Failure_Reason {
					if st == enterprisev1.AuthenticationLog_Entry_FAILURE {
						return enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL
					}
					return enterprisev1.AuthenticationLog_Entry_Failure_REASON_UNSET
				}(),
			})),
		}
	}

	accessLog := func(at time.Duration, sessionRef *metav1.ObjectReference,
		st corev1.AccessLog_Entry_Common_Status) (pendingLog, string) {
		lg := newAccessLog(&accessLogOptions{
			CreatedAt:  base.Add(at),
			SessionRef: sessionRef,
			UserRef:    userRef,
			ServiceRef: randomObjectReference(),
			Status:     st,
		})
		return pendingLog{table: logTableAccess, data: marshalLog(t, lg)}, lg.Metadata.Id
	}

	accUS1, idUS1 := accessLog(2*time.Second, sessUS, corev1.AccessLog_Entry_Common_ALLOWED)
	accUS2, idUS2 := accessLog(3*time.Second, sessUS, corev1.AccessLog_Entry_Common_DENIED)
	accDE, idDE := accessLog(4*time.Second, sessDE, corev1.AccessLog_Entry_Common_ALLOWED)
	accUnknown, idUnknown := accessLog(5*time.Second, sessUnknown, corev1.AccessLog_Entry_Common_ALLOWED)
	accNoGeo, idNoGeo := accessLog(6*time.Second, sessNoGeo, corev1.AccessLog_Entry_Common_ALLOWED)

	require.Nil(t, ts.srv.insertLogBatches(ts.ctx, []*pendingLogBatch{
		{items: []pendingLog{
			authLog(0, sessUS, newTestGeoIP("US", "United States", "New York", 40.7, -74), enterprisev1.AuthenticationLog_Entry_SUCCESS),
			authLog(time.Second, sessDE, newTestGeoIP("de", "Germany", "Berlin", 52.5, 13.4), enterprisev1.AuthenticationLog_Entry_SUCCESS),
			authLog(time.Second, sessUS, newTestGeoIP("RU", "Russia", "Moscow", 55.7, 37.6), enterprisev1.AuthenticationLog_Entry_FAILURE),
			authLog(time.Second, sessNoGeo, nil, enterprisev1.AuthenticationLog_Entry_SUCCESS),
			accUS1,
		}},
		{items: []pendingLog{accUS2, accDE, accUnknown, accNoGeo}},
	}))

	assert.Equal(t, "US", getTestSessionLocation(t, ts.srv, sessUS.Uid).String)
	assert.Equal(t, "DE", getTestSessionLocation(t, ts.srv, sessDE.Uid).String)
	assert.False(t, getTestSessionLocation(t, ts.srv, sessNoGeo.Uid).Valid)
	assert.False(t, getTestSessionLocation(t, ts.srv, sessUnknown.Uid).Valid)

	assert.Equal(t, "US", getTestAccessLogCountry(t, ts.srv, idUS1).String)
	assert.Equal(t, "US", getTestAccessLogCountry(t, ts.srv, idUS2).String)
	assert.Equal(t, "DE", getTestAccessLogCountry(t, ts.srv, idDE).String)
	assert.False(t, getTestAccessLogCountry(t, ts.srv, idUnknown).Valid)
	assert.False(t, getTestAccessLogCountry(t, ts.srv, idNoGeo).Valid)

	accFR, idFR := accessLog(20*time.Second, sessUS, corev1.AccessLog_Entry_Common_ALLOWED)
	require.Nil(t, ts.srv.insertLogBatches(ts.ctx, []*pendingLogBatch{
		{items: []pendingLog{
			authLog(10*time.Second, sessUS, newTestGeoIP("FR", "France", "Paris", 48.8, 2.3), enterprisev1.AuthenticationLog_Entry_SUCCESS),
			authLog(-time.Minute, sessUS, newTestGeoIP("JP", "Japan", "Tokyo", 35.6, 139.6), enterprisev1.AuthenticationLog_Entry_SUCCESS),
			accFR,
		}},
	}))

	assert.Equal(t, "FR", getTestSessionLocation(t, ts.srv, sessUS.Uid).String)
	assert.Equal(t, "FR", getTestAccessLogCountry(t, ts.srv, idFR).String)
	assert.Equal(t, "US", getTestAccessLogCountry(t, ts.srv, idUS1).String)

	require.Nil(t, ts.srv.insertAuthenticationLog(authLog(30*time.Second, sessDE, nil, enterprisev1.AuthenticationLog_Entry_SUCCESS).data))
	assert.False(t, getTestSessionLocation(t, ts.srv, sessDE.Uid).Valid)
	require.Nil(t, ts.srv.insertAccessLog(func() []byte {
		itm, _ := accessLog(31*time.Second, sessDE, corev1.AccessLog_Entry_Common_ALLOWED)
		return itm.data
	}()))

	from := pbutils.Timestamp(base.Add(-time.Hour))
	to := pbutils.Timestamp(base.Add(time.Hour))

	{
		res, err := ts.srv.listAccessLogTopCountry(ts.ctx, &visibilityv1.ListAccessLogTopCountryRequest{
			From: from,
			To:   to,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 3)

		assert.Equal(t, "US", res.Items[0].Country.Code)
		assert.Equal(t, uint64(2), res.Items[0].Count)
		assert.Equal(t, uint64(1), res.Items[0].CountAllowed)
		assert.Equal(t, uint64(1), res.Items[0].CountDenied)
		assert.Equal(t, uint64(1), res.Items[0].CountSession)
		assert.Equal(t, uint64(1), res.Items[0].CountUser)
		assert.Equal(t, uint64(2), res.Items[0].CountService)
		assert.Equal(t, base.Add(3*time.Second).Unix(), res.Items[0].LastSeenAt.AsTime().Unix())

		assert.Equal(t, "DE", res.Items[1].Country.Code)
		assert.Equal(t, uint64(1), res.Items[1].Count)
		assert.Equal(t, "FR", res.Items[2].Country.Code)
		assert.Equal(t, uint64(1), res.Items[2].Count)

		assert.Equal(t, uint64(3), res.TotalCount)
		assert.Equal(t, uint64(0), res.TotalOther)
		assert.Equal(t, uint64(3), res.TotalUnknown)
	}

	{
		res, err := ts.srv.listAccessLogTopCountry(ts.ctx, &visibilityv1.ListAccessLogTopCountryRequest{
			From:   from,
			To:     to,
			Limit:  1,
			Status: corev1.AccessLog_Entry_Common_ALLOWED,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 1)
		assert.Equal(t, uint64(3), res.TotalCount)
		assert.Equal(t, uint64(2), res.TotalOther)
		assert.Equal(t, uint64(1), res.Items[0].Count)
	}

	{
		res, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			Common:      &vmetav1.CommonListOptions{},
			From:        from,
			To:          to,
			CountryCode: "us",
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(2), res.ListResponseMeta.TotalCount)
		for _, itm := range res.Items {
			assert.Equal(t, sessUS.Uid, itm.Entry.Common.SessionRef.Uid)
		}
	}

	{
		res, err := ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			From: from,
			To:   to,
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(7), res.TotalNumber)
		assert.Equal(t, uint64(3), res.TotalCountry)

		res, err = ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			From:        from,
			To:          to,
			CountryCode: "DE",
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), res.TotalNumber)
		assert.Equal(t, uint64(1), res.TotalCountry)
	}

	{
		res, err := ts.srv.getAccessLogDataPoint(ts.ctx, &visibilityv1.GetAccessLogDataPointRequest{
			From:    from,
			To:      to,
			GroupBy: visibilityv1.GetAccessLogDataPointRequest_COUNTRY,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Series, 3)
		assert.Equal(t, "US", res.Series[0].Key)
		assert.Equal(t, int64(2), res.Series[0].Total)
	}

	{
		res, err := ts.srv.listAccessLogTopUser(ts.ctx, &visibilityv1.ListAccessLogTopUserRequest{
			From:        from,
			To:          to,
			CountryCode: "FR",
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(1), res.TotalCount)
	}

	for _, code := range []string{"USA", "1A", "U'"} {
		_, err := ts.srv.listAccessLog(ts.ctx, &visibilityv1.ListAccessLogRequest{
			Common:      &vmetav1.CommonListOptions{},
			CountryCode: code,
		})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), code)

		_, err = ts.srv.getSummaryAccessLog(ts.ctx, &visibilityv1.GetAccessLogSummaryRequest{
			CountryCode: code,
		})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), code)
	}
}

func TestAuthenticationLogGeoQueries(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	user1 := randomObjectReference()
	user2 := randomObjectReference()
	sess1 := randomObjectReference()
	sess2 := randomObjectReference()
	sess3 := randomObjectReference()

	idx := 0
	insert := func(opts *authenticationLogOptions) {
		opts.CreatedAt = base.Add(time.Duration(idx) * time.Second)
		if opts.Type == corev1.Session_Status_Authentication_Info_TYPE_UNSET {
			opts.Type = corev1.Session_Status_Authentication_Info_IDENTITY_PROVIDER
		}
		if opts.Status == enterprisev1.AuthenticationLog_Entry_FAILURE {
			opts.FailureReason = enterprisev1.AuthenticationLog_Entry_Failure_INVALID_CREDENTIAL
		}
		require.Nil(t, ts.srv.insertAuthenticationLog(marshalLog(t, newAuthenticationLog(opts))))
		idx++
	}

	for i := range 4 {
		insert(&authenticationLogOptions{
			UserRef:    user1,
			SessionRef: sess1,
			GeoIP:      newTestGeoIP("DE", "Germany", "Berlin", 52+float64(i)*0.1, 13),
		})
	}
	insert(&authenticationLogOptions{
		UserRef:    user2,
		SessionRef: sess2,
		GeoIP:      newTestGeoIP("DE", "Germany", "Munich", 48.1, 11.5),
	})
	for range 3 {
		insert(&authenticationLogOptions{
			Status: enterprisev1.AuthenticationLog_Entry_FAILURE,
			GeoIP:  newTestGeoIP("BR", "Brazil", "São Paulo", -23.5, -46.6),
		})
	}
	insert(&authenticationLogOptions{
		UserRef:    user2,
		SessionRef: sess3,
		GeoIP:      newTestGeoIP("BR", "Brazil", "", -10, -55),
	})
	insert(&authenticationLogOptions{
		UserRef:    user2,
		SessionRef: sess3,
		GeoIP: &corev1.GeoIP{
			Country: &corev1.GeoIP_Country{Code: "SG", Name: "Singapore"},
		},
	})
	for range 2 {
		insert(&authenticationLogOptions{
			UserRef:    user1,
			SessionRef: sess1,
		})
	}

	from := pbutils.Timestamp(base.Add(-time.Minute))
	to := pbutils.Timestamp(base.Add(time.Hour))

	{
		res, err := ts.srv.listAuthenticationLogTopCountry(ts.ctx, &visibilityv1.ListAuthenticationLogTopCountryRequest{
			From: from,
			To:   to,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 3)

		assert.Equal(t, "DE", res.Items[0].Country.Code)
		assert.Equal(t, "Germany", res.Items[0].Country.Name)
		assert.Equal(t, "EU", res.Items[0].Continent.Code)
		assert.Equal(t, uint64(5), res.Items[0].Count)
		assert.Equal(t, uint64(5), res.Items[0].CountSuccess)
		assert.Equal(t, uint64(0), res.Items[0].CountFailure)
		assert.Equal(t, uint64(2), res.Items[0].CountUser)
		assert.Equal(t, uint64(2), res.Items[0].CountSession)
		assert.Equal(t, base.Add(4*time.Second).Unix(), res.Items[0].LastSeenAt.AsTime().Unix())

		assert.Equal(t, "BR", res.Items[1].Country.Code)
		assert.Equal(t, uint64(4), res.Items[1].Count)
		assert.Equal(t, uint64(1), res.Items[1].CountSuccess)
		assert.Equal(t, uint64(3), res.Items[1].CountFailure)
		assert.Equal(t, uint64(1), res.Items[1].CountUser)

		assert.Equal(t, "SG", res.Items[2].Country.Code)
		assert.Nil(t, res.Items[2].Continent)

		assert.Equal(t, uint64(3), res.TotalCount)
		assert.Equal(t, uint64(0), res.TotalOther)
		assert.Equal(t, uint64(2), res.TotalUnknown)
	}

	{
		res, err := ts.srv.listAuthenticationLogTopCountry(ts.ctx, &visibilityv1.ListAuthenticationLogTopCountryRequest{
			From:   from,
			To:     to,
			Status: enterprisev1.AuthenticationLog_Entry_FAILURE,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 1)
		assert.Equal(t, "BR", res.Items[0].Country.Code)
		assert.Equal(t, uint64(3), res.Items[0].Count)
		assert.Equal(t, uint64(0), res.TotalUnknown)

		res, err = ts.srv.listAuthenticationLogTopCountry(ts.ctx, &visibilityv1.ListAuthenticationLogTopCountryRequest{
			From:    from,
			To:      to,
			UserRef: user1,
			Limit:   1,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 1)
		assert.Equal(t, "DE", res.Items[0].Country.Code)
		assert.Equal(t, uint64(4), res.Items[0].Count)
		assert.Equal(t, uint64(2), res.TotalUnknown)
	}

	{
		res, err := ts.srv.listAuthenticationLogTopCity(ts.ctx, &visibilityv1.ListAuthenticationLogTopCityRequest{
			From: from,
			To:   to,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 4)

		assert.Equal(t, "Berlin", res.Items[0].City.Name)
		assert.Equal(t, "DE", res.Items[0].Country.Code)
		assert.Equal(t, "Berlin region", res.Items[0].Region.Name)
		assert.InDelta(t, 52.15, res.Items[0].Coordinates.Latitude, 0.0001)
		assert.InDelta(t, 13, res.Items[0].Coordinates.Longitude, 0.0001)
		assert.Equal(t, float64(20), res.Items[0].Coordinates.AccuracyRadius)
		assert.Equal(t, uint64(4), res.Items[0].Count)
		assert.Equal(t, uint64(1), res.Items[0].CountSession)

		assert.Equal(t, "São Paulo", res.Items[1].City.Name)
		assert.Equal(t, uint64(3), res.Items[1].CountFailure)

		assert.Equal(t, "BR", res.Items[2].Country.Code)
		assert.Nil(t, res.Items[2].City)
		assert.Nil(t, res.Items[2].Region)
		assert.Equal(t, uint64(1), res.Items[2].Count)

		assert.Equal(t, "Munich", res.Items[3].City.Name)

		assert.Equal(t, uint64(4), res.TotalCount)
		assert.Equal(t, uint64(3), res.TotalUnknown)

		res, err = ts.srv.listAuthenticationLogTopCity(ts.ctx, &visibilityv1.ListAuthenticationLogTopCityRequest{
			From:        from,
			To:          to,
			CountryCode: "de",
			Limit:       1,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Items, 1)
		assert.Equal(t, "Berlin", res.Items[0].City.Name)
		assert.Equal(t, uint64(2), res.TotalCount)
		assert.Equal(t, uint64(1), res.TotalOther)
		assert.Equal(t, uint64(0), res.TotalUnknown)
	}

	{
		res, err := ts.srv.listAuthenticationLog(ts.ctx, &visibilityv1.ListAuthenticationLogRequest{
			Common:      &vmetav1.CommonListOptions{},
			From:        from,
			To:          to,
			CountryCode: "BR",
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint32(4), res.ListResponseMeta.TotalCount)
		for _, itm := range res.Items {
			assert.Equal(t, "BR", itm.Entry.Authentication.Info.Geoip.Country.Code)
		}
	}

	{
		res, err := ts.srv.getSummaryAuthenticationLog(ts.ctx, &visibilityv1.GetAuthenticationLogSummaryRequest{
			From: from,
			To:   to,
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(12), res.TotalNumber)
		assert.Equal(t, uint64(3), res.TotalCountry)

		res, err = ts.srv.getSummaryAuthenticationLog(ts.ctx, &visibilityv1.GetAuthenticationLogSummaryRequest{
			From:        from,
			To:          to,
			CountryCode: "BR",
			Status:      enterprisev1.AuthenticationLog_Entry_FAILURE,
		})
		require.Nil(t, err, "%+v", err)
		assert.Equal(t, uint64(3), res.TotalNumber)
		assert.Equal(t, uint64(1), res.TotalCountry)
	}

	{
		res, err := ts.srv.getAuthenticationLogDataPoint(ts.ctx, &visibilityv1.GetAuthenticationLogDataPointRequest{
			From:    from,
			To:      to,
			GroupBy: visibilityv1.GetAuthenticationLogDataPointRequest_COUNTRY,
		})
		require.Nil(t, err, "%+v", err)
		require.Len(t, res.Series, 3)
		assert.Equal(t, "DE", res.Series[0].Key)
		assert.Equal(t, int64(5), res.Series[0].Total)
	}

	{
		_, err := ts.srv.listAuthenticationLogTopUser(ts.ctx, &visibilityv1.ListAuthenticationLogTopUserRequest{
			From:        from,
			To:          to,
			CountryCode: "Germany",
		})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))

		_, err = ts.srv.listAuthenticationLogTopCity(ts.ctx, &visibilityv1.ListAuthenticationLogTopCityRequest{
			CountryCode: "D",
		})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	}
}

func TestSetSessionLocations(t *testing.T) {
	ts := newTestServer(t)
	require.NotNil(t, ts)

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)

	sess1 := randomObjectReference()
	sess2 := randomObjectReference()

	for _, opts := range []*authenticationLogOptions{
		{CreatedAt: base, SessionRef: sess1, GeoIP: newTestGeoIP("US", "United States", "", 37, -95)},
		{CreatedAt: base.Add(2 * time.Second), SessionRef: sess1, GeoIP: newTestGeoIP("ca", "Canada", "", 56, -106)},
		{
			CreatedAt: base.Add(3 * time.Second), SessionRef: sess1,
			Status: enterprisev1.AuthenticationLog_Entry_FAILURE, GeoIP: newTestGeoIP("RU", "Russia", "", 61, 105),
		},
		{CreatedAt: base.Add(time.Second), SessionRef: sess2},
		{CreatedAt: base.Add(time.Second), GeoIP: newTestGeoIP("IT", "Italy", "", 41, 12)},
	} {
		insertLogJSON(t, ts.srv, "authentication_logs", marshalLog(t, newAuthenticationLog(opts)))
	}

	assert.Equal(t, 0, getTableCount(t, ts.srv, tableSessionLocations))

	require.Nil(t, ts.srv.setSessionLocations(ts.ctx))
	assert.Equal(t, 2, getTableCount(t, ts.srv, tableSessionLocations))
	assert.Equal(t, "CA", getTestSessionLocation(t, ts.srv, sess1.Uid).String)
	assert.False(t, getTestSessionLocation(t, ts.srv, sess2.Uid).Valid)

	_, err := ts.srv.db.Exec(fmt.Sprintf(`UPDATE %s SET %s = 'FR'`, tableSessionLocations, colCountryCode))
	require.Nil(t, err)
	require.Nil(t, ts.srv.setSessionLocations(ts.ctx))
	assert.Equal(t, "FR", getTestSessionLocation(t, ts.srv, sess1.Uid).String)

	_, err = ts.srv.db.Exec(fmt.Sprintf(`UPDATE %s SET updated_at = ? WHERE %s = ?`,
		tableSessionLocations, colSessionUID), time.Now().UTC().Add(-60*24*time.Hour), sess2.Uid)
	require.Nil(t, err)

	require.Nil(t, ts.srv.doCleanup(ts.ctx))
	assert.Equal(t, 1, getTableCount(t, ts.srv, tableSessionLocations))
	assert.Equal(t, "FR", getTestSessionLocation(t, ts.srv, sess1.Uid).String)

	require.Nil(t, ts.srv.initDB(ts.ctx))
	assert.Equal(t, 1, getTableCount(t, ts.srv, tableSessionLocations))
}
