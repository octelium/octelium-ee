// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package logstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1"
	"github.com/octelium/octelium/cluster/common/apivalidation"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

const tableSessionLocations = "session_locations"

const maxGeoTopItems = 250

const jsonAuthenticationGeoIP = `$.entry.authentication.info.geoip`

var sqlAuthenticationCountryCode = fmt.Sprintf(
	`NULLIF(json_extract_string(rsc, '%s.country.code'), '')`, jsonAuthenticationGeoIP)

var sqlAuthenticationCityName = fmt.Sprintf(
	`COALESCE(json_extract_string(rsc, '%s.city.name'), '')`, jsonAuthenticationGeoIP)

var sqlAuthenticationLatitude = fmt.Sprintf(
	`COALESCE(TRY_CAST(json_extract_string(rsc, '%s.coordinates.latitude') AS DOUBLE), 0)`,
	jsonAuthenticationGeoIP)

var sqlAuthenticationLongitude = fmt.Sprintf(
	`COALESCE(TRY_CAST(json_extract_string(rsc, '%s.coordinates.longitude') AS DOUBLE), 0)`,
	jsonAuthenticationGeoIP)

var sqlAuthenticationAccuracyRadius = fmt.Sprintf(
	`COALESCE(TRY_CAST(json_extract_string(rsc, '%s.coordinates.accuracyRadius') AS DOUBLE), 0)`,
	jsonAuthenticationGeoIP)

var sqlAuthenticationHasCoordinates = fmt.Sprintf(
	`(json_extract(rsc, '%s.coordinates') IS NOT NULL AND (%s <> 0 OR %s <> 0))`,
	jsonAuthenticationGeoIP, sqlAuthenticationLatitude, sqlAuthenticationLongitude)

var sqlAuthenticationCityKey = fmt.Sprintf(
	`CASE WHEN %s THEN CONCAT(COALESCE(%s, ''), '/', %s) END`,
	sqlAuthenticationHasCoordinates, sqlAuthenticationCountryCode, sqlAuthenticationCityName)

const jsonAuthenticationUserUID = `json_extract_string(rsc, '$.entry.userRef.uid')`
const jsonAuthenticationSessionUID = `json_extract_string(rsc, '$.entry.sessionRef.uid')`

func getGeoTopLimit(limit uint32) int {
	switch {
	case limit == 0:
		return defaultTopItems
	case limit > maxGeoTopItems:
		return maxGeoTopItems
	default:
		return int(limit)
	}
}

func getCountryCode(code string) (string, error) {
	ret := strings.ToUpper(strings.TrimSpace(code))
	if len(ret) != 2 || ret[0] < 'A' || ret[0] > 'Z' || ret[1] < 'A' || ret[1] > 'Z' {
		return "", grpcutils.InvalidArg("Invalid country code: %s", code)
	}

	return ret, nil
}

func appendAccessCountryFilter(filters []exp.Expression, code string) ([]exp.Expression, error) {
	if code == "" {
		return filters, nil
	}

	code, err := getCountryCode(code)
	if err != nil {
		return nil, err
	}

	return append(filters, goqu.L(colCountryCode).Eq(code)), nil
}

func appendAuthenticationCountryFilter(filters []exp.Expression, code string) ([]exp.Expression, error) {
	if code == "" {
		return filters, nil
	}

	code, err := getCountryCode(code)
	if err != nil {
		return nil, err
	}

	return append(filters, goqu.L(sqlAuthenticationCountryCode).Eq(code)), nil
}

func appendAuthenticationRefFilters(filters []exp.Expression,
	sessionRef, userRef, deviceRef, identityProviderRef,
	credentialRef, authenticatorRef *metav1.ObjectReference) ([]exp.Expression, error) {

	var err error

	for _, arg := range []struct {
		ref *metav1.ObjectReference
		pth string
	}{
		{ref: userRef, pth: "entry.userRef"},
		{ref: deviceRef, pth: "entry.deviceRef"},
		{ref: sessionRef, pth: "entry.sessionRef"},
		{ref: identityProviderRef, pth: "entry.authentication.info.identityProvider.identityProviderRef"},
		{ref: authenticatorRef, pth: "entry.authentication.info.authenticator.authenticatorRef"},
		{ref: credentialRef, pth: "entry.authentication.info.credential.credentialRef"},
	} {
		filters, err = appendRefFilter(filters, arg.ref, nil, "", arg.pth)
		if err != nil {
			return nil, err
		}
	}

	return filters, nil
}

func getSessionLocationUpsertQuery() string {
	return fmt.Sprintf(`
INSERT INTO %s (%s, %s, updated_at)
SELECT uid, code, ts FROM (
	SELECT
		json_extract_string(j, '$.entry.sessionRef.uid') AS uid,
		NULLIF(upper(json_extract_string(j, '%s.country.code')), '') AS code,
		TRY_CAST(json_extract_string(j, '$.metadata.createdAt') AS TIMESTAMP) AS ts,
		COALESCE(json_extract_string(j, '$.entry.status'), 'SUCCESS') AS status
	FROM (SELECT CAST($1 AS JSON) AS j)
) WHERE uid IS NOT NULL AND uid <> '' AND ts IS NOT NULL AND status = 'SUCCESS'
ON CONFLICT (%s) DO UPDATE SET %s = excluded.%s, updated_at = excluded.updated_at
WHERE excluded.updated_at >= %s.updated_at`,
		tableSessionLocations, colSessionUID, colCountryCode,
		jsonAuthenticationGeoIP,
		colSessionUID, colCountryCode, colCountryCode, tableSessionLocations)
}

func (s *Server) initSessionLocations(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (%s VARCHAR PRIMARY KEY, %s VARCHAR, updated_at TIMESTAMP)`,
		tableSessionLocations, colSessionUID, colCountryCode))

	return err
}

func (s *Server) setSessionLocations(ctx context.Context) error {
	var count int64
	if err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM %s`, tableSessionLocations)).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s (%s, %s, updated_at)
SELECT uid, code, %s FROM (
	SELECT %s AS uid, upper(%s) AS code, %s
	FROM authentication_logs
	WHERE %s = 'SUCCESS' AND %s IS NOT NULL
) WHERE uid IS NOT NULL AND uid <> ''
QUALIFY row_number() OVER (PARTITION BY uid ORDER BY %s DESC) = 1`,
		tableSessionLocations, colSessionUID, colCountryCode, colCreatedAt,
		jsonAuthenticationSessionUID, sqlAuthenticationCountryCode, colCreatedAt,
		sqlAuthenticationStatus, colCreatedAt, colCreatedAt))

	return err
}

type geoTopTotals struct {
	count   uint64
	sum     uint64
	unknown uint64
}

func (t *geoTopTotals) other(returned uint64) uint64 {
	if t.sum <= returned {
		return 0
	}

	return t.sum - returned
}

func (s *Server) getGeoTopTotals(ctx context.Context,
	table, keyExpr string, filters []exp.Expression) (*geoTopTotals, error) {

	ds := goqu.Dialect("postgres").From(table).Where(filters...).Select(
		goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, keyExpr)),
		goqu.L(fmt.Sprintf(`COUNT(%s)`, keyExpr)),
		goqu.L(fmt.Sprintf(`COUNT(*) - COUNT(%s)`, keyExpr)),
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}

	ret := &geoTopTotals{}
	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&ret.count, &ret.sum, &ret.unknown); err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Server) listAuthenticationLogTopCountry(ctx context.Context,
	req *visibilityv1.ListAuthenticationLogTopCountryRequest) (*visibilityv1.ListAuthenticationLogTopCountryResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopCountryResponse{}

	filters, err := appendAuthenticationRefFilters(nil, req.SessionRef, req.UserRef, req.DeviceRef,
		req.IdentityProviderRef, req.CredentialRef, req.AuthenticatorRef)
	if err != nil {
		return nil, err
	}

	filters = appendAuthenticationStatusFilters(filters, req.Status, req.Reason)
	filters = appendTimeFilters(filters, req.From, req.To)

	ds := goqu.Dialect("postgres").From("authentication_logs").
		Select(
			goqu.L(sqlAuthenticationCountryCode).As("code"),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(json_extract_string(rsc, '%s.country.name')), '')`,
				jsonAuthenticationGeoIP)),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(json_extract_string(rsc, '%s.continent.code')), '')`,
				jsonAuthenticationGeoIP)),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(json_extract_string(rsc, '%s.continent.name')), '')`,
				jsonAuthenticationGeoIP)),
			goqu.L(`COUNT(*)`).As("count"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUCCESS')`, sqlAuthenticationStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FAILURE')`, sqlAuthenticationStatus)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, jsonAuthenticationUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, jsonAuthenticationSessionUID)),
			goqu.L(fmt.Sprintf(`MAX(%s)`, colCreatedAt)),
		).
		Where(filters...).
		Where(goqu.L(fmt.Sprintf(`%s IS NOT NULL`, sqlAuthenticationCountryCode))).
		GroupBy(goqu.L("code")).
		Order(goqu.L("count").Desc(), goqu.L("code").Asc()).
		Limit(uint(getGeoTopLimit(req.Limit)))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var returned uint64

	for rows.Next() {
		var code, name, continentCode, continentName string
		var lastSeenAt sql.NullTime
		item := &visibilityv1.ListAuthenticationLogTopCountryResponse_Item{}

		if err := rows.Scan(&code, &name, &continentCode, &continentName,
			&item.Count, &item.CountSuccess, &item.CountFailure,
			&item.CountUser, &item.CountSession, &lastSeenAt); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		item.Country = &corev1.GeoIP_Country{Code: code, Name: name}
		if continentCode != "" || continentName != "" {
			item.Continent = &corev1.GeoIP_Continent{Code: continentCode, Name: continentName}
		}
		if lastSeenAt.Valid {
			item.LastSeenAt = pbutils.Timestamp(lastSeenAt.Time)
		}

		ret.Items = append(ret.Items, item)
		returned = returned + item.Count
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	totals, err := s.getGeoTopTotals(ctx, "authentication_logs", sqlAuthenticationCountryCode, filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalCount = totals.count
	ret.TotalOther = totals.other(returned)
	ret.TotalUnknown = totals.unknown

	return ret, nil
}

func (s *Server) listAuthenticationLogTopCity(ctx context.Context,
	req *visibilityv1.ListAuthenticationLogTopCityRequest) (*visibilityv1.ListAuthenticationLogTopCityResponse, error) {

	ret := &visibilityv1.ListAuthenticationLogTopCityResponse{}

	filters, err := appendAuthenticationRefFilters(nil, req.SessionRef, req.UserRef, req.DeviceRef,
		req.IdentityProviderRef, req.CredentialRef, req.AuthenticatorRef)
	if err != nil {
		return nil, err
	}

	filters = appendAuthenticationStatusFilters(filters, req.Status, req.Reason)

	filters, err = appendAuthenticationCountryFilter(filters, req.CountryCode)
	if err != nil {
		return nil, err
	}

	filters = appendTimeFilters(filters, req.From, req.To)

	ds := goqu.Dialect("postgres").From("authentication_logs").
		Select(
			goqu.L(fmt.Sprintf(`COALESCE(%s, '')`, sqlAuthenticationCountryCode)).As("code"),
			goqu.L(sqlAuthenticationCityName).As("city"),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(json_extract_string(rsc, '%s.country.name')), '')`,
				jsonAuthenticationGeoIP)),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(json_extract_string(rsc, '%s.region.code')), '')`,
				jsonAuthenticationGeoIP)),
			goqu.L(fmt.Sprintf(`COALESCE(any_value(json_extract_string(rsc, '%s.region.name')), '')`,
				jsonAuthenticationGeoIP)),
			goqu.L(fmt.Sprintf(`AVG(%s)`, sqlAuthenticationLatitude)),
			goqu.L(fmt.Sprintf(`AVG(%s)`, sqlAuthenticationLongitude)),
			goqu.L(fmt.Sprintf(`MAX(%s)`, sqlAuthenticationAccuracyRadius)),
			goqu.L(`COUNT(*)`).As("count"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUCCESS')`, sqlAuthenticationStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FAILURE')`, sqlAuthenticationStatus)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, jsonAuthenticationUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, jsonAuthenticationSessionUID)),
			goqu.L(fmt.Sprintf(`MAX(%s)`, colCreatedAt)),
		).
		Where(filters...).
		Where(goqu.L(sqlAuthenticationHasCoordinates)).
		GroupBy(goqu.L("code"), goqu.L("city")).
		Order(goqu.L("count").Desc(), goqu.L("code").Asc(), goqu.L("city").Asc()).
		Limit(uint(getGeoTopLimit(req.Limit)))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var returned uint64

	for rows.Next() {
		var code, city, name, regionCode, regionName string
		var lastSeenAt sql.NullTime
		coordinates := &corev1.GeoIP_Coordinates{}
		item := &visibilityv1.ListAuthenticationLogTopCityResponse_Item{}

		if err := rows.Scan(&code, &city, &name, &regionCode, &regionName,
			&coordinates.Latitude, &coordinates.Longitude, &coordinates.AccuracyRadius,
			&item.Count, &item.CountSuccess, &item.CountFailure,
			&item.CountUser, &item.CountSession, &lastSeenAt); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		if code != "" || name != "" {
			item.Country = &corev1.GeoIP_Country{Code: code, Name: name}
		}
		if regionCode != "" || regionName != "" {
			item.Region = &corev1.GeoIP_Region{Code: regionCode, Name: regionName}
		}
		if city != "" {
			item.City = &corev1.GeoIP_City{Name: city}
		}
		item.Coordinates = coordinates
		if lastSeenAt.Valid {
			item.LastSeenAt = pbutils.Timestamp(lastSeenAt.Time)
		}

		ret.Items = append(ret.Items, item)
		returned = returned + item.Count
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	totals, err := s.getGeoTopTotals(ctx, "authentication_logs", sqlAuthenticationCityKey, filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalCount = totals.count
	ret.TotalOther = totals.other(returned)
	ret.TotalUnknown = totals.unknown

	return ret, nil
}

func (s *Server) listAccessLogTopCountry(ctx context.Context,
	req *visibilityv1.ListAccessLogTopCountryRequest) (*visibilityv1.ListAccessLogTopCountryResponse, error) {

	ret := &visibilityv1.ListAccessLogTopCountryResponse{}

	var filters []exp.Expression
	var err error

	filters, err = appendRefFilter(filters, req.UserRef, nil, colUserUID, "entry.common.userRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.DeviceRef, nil, colDeviceUID, "entry.common.deviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.SessionRef, nil, colSessionUID, "entry.common.sessionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.ServiceRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMust: 1,
	}, colServiceUID, "entry.common.serviceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.NamespaceRef, nil, colNamespaceUID, "entry.common.namespaceRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.RegionRef, nil, colRegionUID, "entry.common.regionRef")
	if err != nil {
		return nil, err
	}
	filters, err = appendRefFilter(filters, req.PolicyRef, &apivalidation.CheckGetOptionsOpts{
		ParentsMax: 8,
	}, colPolicyUID, "entry.common.reason.details.policyMatch.policy.policyRef")
	if err != nil {
		return nil, err
	}

	filters = appendAccessStatusFilter(filters, req.Status)

	filters = appendAccessFlagFilters(filters, req.IsPublic, req.IsAnonymous)

	filters = appendAccessReasonModeFilters(filters, req.Reason, req.Mode)

	filters = appendTimeFilters(filters, req.From, req.To)

	ds := goqu.Dialect("postgres").From("access_logs").
		Select(
			goqu.L(colCountryCode).As("code"),
			goqu.L(`COUNT(*)`).As("count"),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ALLOWED')`, colStatus)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DENIED')`, colStatus)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, colSessionUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s)`, colServiceUID)),
			goqu.L(fmt.Sprintf(`MAX(%s)`, colCreatedAt)),
		).
		Where(filters...).
		Where(goqu.L(fmt.Sprintf(`%s IS NOT NULL`, colCountryCode))).
		GroupBy(goqu.L("code")).
		Order(goqu.L("count").Desc(), goqu.L("code").Asc()).
		Limit(uint(getGeoTopLimit(req.Limit)))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var returned uint64

	for rows.Next() {
		var code string
		var lastSeenAt sql.NullTime
		item := &visibilityv1.ListAccessLogTopCountryResponse_Item{}

		if err := rows.Scan(&code, &item.Count, &item.CountAllowed, &item.CountDenied,
			&item.CountUser, &item.CountSession, &item.CountService, &lastSeenAt); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		item.Country = &corev1.GeoIP_Country{Code: code}
		if lastSeenAt.Valid {
			item.LastSeenAt = pbutils.Timestamp(lastSeenAt.Time)
		}

		ret.Items = append(ret.Items, item)
		returned = returned + item.Count
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	totals, err := s.getGeoTopTotals(ctx, "access_logs", colCountryCode, filters)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalCount = totals.count
	ret.TotalOther = totals.other(returned)
	ret.TotalUnknown = totals.unknown

	return ret, nil
}
