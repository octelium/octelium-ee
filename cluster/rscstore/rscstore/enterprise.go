// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package rscstore

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/venterprisev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
)

func (s *Server) getSummaryEnterpriseCollectorExporter(ctx context.Context, req *venterprisev1.GetCollectorExporterSummaryRequest) (*venterprisev1.GetCollectorExporterSummaryResponse, error) {

	ret := &venterprisev1.GetCollectorExporterSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindCollectorExporter))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_disabled`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.otlp') IS NOT NULL) AS count_otlp`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.otlpHTTP') IS NOT NULL) AS count_otlp_http`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.clickhouse') IS NOT NULL) AS count_clickhouse`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.elasticsearch') IS NOT NULL) AS count_elasticsearch`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.logzio') IS NOT NULL) AS count_logzio`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.influxDB') IS NOT NULL) AS count_influxdb`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.kafka') IS NOT NULL) AS count_kafka`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.datadog') IS NOT NULL) AS count_datadog`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.splunk') IS NOT NULL) AS count_splunk`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.azureMonitor') IS NOT NULL) AS count_azure_monitor`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.azureDataExplorer') IS NOT NULL) AS count_azure_data_explorer`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.prometheusRemoteWrite') IS NOT NULL) AS count_prometheus_remote_write`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalDisabled,
			&ret.TotalOTLP, &ret.TotalOTLPHTTP, &ret.TotalClickhouse, &ret.TotalElasticsearch,
			&ret.TotalLogzio, &ret.TotalInfluxDB, &ret.TotalKafka, &ret.TotalDatadog, &ret.TotalSplunk,
			&ret.TotalAzureMonitor, &ret.TotalAzureDataExplorer, &ret.TotalPrometheusRemoteWrite)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseSecret(ctx context.Context, req *venterprisev1.GetSecretSummaryRequest) (*venterprisev1.GetSecretSummaryResponse, error) {

	ret := &venterprisev1.GetSecretSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindSecret))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseSecretStore(ctx context.Context, req *venterprisev1.GetSecretStoreSummaryRequest) (*venterprisev1.GetSecretStoreSummaryResponse, error) {

	ret := &venterprisev1.GetSecretStoreSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindSecretStore))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'TYPE_AZURE_KEY_VAULT') AS count_azure_key_vault`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'TYPE_HASHICORP_VAULT') AS count_hashicorp_vault`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'TYPE_GCP_KMS') AS count_gcp_kms`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'TYPE_AWS_KMS') AS count_aws_kms`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'KUBERNETES') AS count_kubernetes`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'OK') AS count_ok`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'LOADING') AS count_loading`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.synchronization.state' = 'SYNCING') AS count_synchronizing`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.synchronization.state' = 'SUCCESS') AS count_sync_success`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.synchronization.state' = 'FAILED') AS count_sync_failed`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalAzureKeyVault, &ret.TotalHashicorpVault, &ret.TotalGCPKMS, &ret.TotalAWSKMS, &ret.TotalKubernetes,
			&ret.TotalOK, &ret.TotalLoading,
			&ret.TotalSynchronizing, &ret.TotalSynchronizationSuccess, &ret.TotalSynchronizationFailed)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseCertificate(ctx context.Context, req *venterprisev1.GetCertificateSummaryRequest) (*venterprisev1.GetCertificateSummaryResponse, error) {

	ret := &venterprisev1.GetCertificateSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindCertificate))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)
	expiringSoonStr := now.Add(30 * 24 * time.Hour).Format(time.RFC3339Nano)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.mode' = 'MANAGED') AS count_managed`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.mode' = 'MANUAL') AS count_manual`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.issuance.state' = 'ISSUANCE_REQUESTED') AS count_issuance_requested`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.issuance.state' = 'ISSUING') AS count_issuing`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.issuance.state' = 'SUCCESS') AS count_issuance_success`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.issuance.state' = 'FAILED') AS count_issuance_failed`),
			goqu.L(`COUNT(*) FILTER (WHERE (rsc->>'$.status.issuance.expiresAt') < ?) AS count_expired`, nowStr),
			goqu.L(`COUNT(*) FILTER (WHERE (rsc->>'$.status.issuance.expiresAt' >= ?) AND (rsc->>'$.status.issuance.expiresAt' < ?)) AS count_expiring_soon`,
				nowStr, expiringSoonStr),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.status.serviceRef') IS NOT NULL) AS count_service`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.status.namespaceRef') IS NOT NULL) AS count_namespace`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.certificateIssuerRef.uid')) AS count_certificate_issuer`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalManaged, &ret.TotalManual,
			&ret.TotalIssuanceRequested, &ret.TotalIssuing, &ret.TotalIssuanceSuccess, &ret.TotalIssuanceFailed,
			&ret.TotalExpired, &ret.TotalExpiringSoon,
			&ret.TotalService, &ret.TotalNamespace, &ret.TotalCertificateIssuer)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseCertificateIssuer(ctx context.Context, req *venterprisev1.GetCertificateIssuerSummaryRequest) (*venterprisev1.GetCertificateIssuerSummaryResponse, error) {

	ret := &venterprisev1.GetCertificateIssuerSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindCertificateIssuer))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.acme') IS NOT NULL) AS count_acme`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'PREPARING') AS count_preparing`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'READY') AS count_ready`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'NOT_READY') AS count_not_ready`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalACME, &ret.TotalPreparing, &ret.TotalReady, &ret.TotalNotReady)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseDNSProvider(ctx context.Context, req *venterprisev1.GetDNSProviderSummaryRequest) (*venterprisev1.GetDNSProviderSummaryResponse, error) {

	ret := &venterprisev1.GetDNSProviderSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindDNSProvider))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.cloudflare') IS NOT NULL) AS count_cloudflare`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.aws') IS NOT NULL) AS count_aws`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.digitalocean') IS NOT NULL) AS count_digitalocean`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.google') IS NOT NULL) AS count_google`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.azure') IS NOT NULL) AS count_azure`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.linode') IS NOT NULL) AS count_linode`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.ovh') IS NOT NULL) AS count_ovh`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalCloudflare, &ret.TotalAWS, &ret.TotalDigitalOcean,
			&ret.TotalGoogle, &ret.TotalAzure, &ret.TotalLinode, &ret.TotalOVH)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseDirectoryProvider(ctx context.Context, req *venterprisev1.GetDirectoryProviderSummaryRequest) (*venterprisev1.GetDirectoryProviderSummaryResponse, error) {

	ret := &venterprisev1.GetDirectoryProviderSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindDirectoryProvider))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_disabled`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.scim') IS NOT NULL) AS count_scim`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.googleWorkspace') IS NOT NULL) AS count_google_workspace`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.keycloak') IS NOT NULL) AS count_keycloak`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.synchronization.state' = 'SYNCING') AS count_synchronizing`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.synchronization.state' = 'SUCCESS') AS count_sync_success`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.synchronization.state' = 'FAILED') AS count_sync_failed`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalDisabled,
			&ret.TotalSCIM, &ret.TotalGoogleWorkspace, &ret.TotalKeycloak,
			&ret.TotalSynchronizing, &ret.TotalSynchronizationSuccess, &ret.TotalSynchronizationFailed)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var userFilters []exp.Expression
	{
		userFilters = append(userFilters, goqu.L(`kind`).Eq(uenterprisev1.KindDirectoryProviderUser))
		userFilters = append(userFilters, goqu.L(`api`).Eq(uenterprisev1.API))
		userFilters = append(userFilters, goqu.L(`version`).Eq(uenterprisev1.Version))
		userFilters = append(userFilters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	userDS := goqu.From("resources").Where(userFilters...).Select(goqu.L(`COUNT(*) AS count_total`))

	userSqln, userSqlargs, err := userDS.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	userRows, err := s.db.QueryContext(ctx, userSqln, userSqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer userRows.Close()

	for userRows.Next() {
		if err := userRows.Scan(&ret.TotalUser); err != nil {
			return nil, err
		}
	}
	if err := userRows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	var groupFilters []exp.Expression
	{
		groupFilters = append(groupFilters, goqu.L(`kind`).Eq(uenterprisev1.KindDirectoryProviderGroup))
		groupFilters = append(groupFilters, goqu.L(`api`).Eq(uenterprisev1.API))
		groupFilters = append(groupFilters, goqu.L(`version`).Eq(uenterprisev1.Version))
		groupFilters = append(groupFilters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	groupDS := goqu.From("resources").Where(groupFilters...).Select(goqu.L(`COUNT(*) AS count_total`))

	groupSqln, groupSqlargs, err := groupDS.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	groupRows, err := s.db.QueryContext(ctx, groupSqln, groupSqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		if err := groupRows.Scan(&ret.TotalGroup); err != nil {
			return nil, err
		}
	}
	if err := groupRows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseDirectoryProviderUser(ctx context.Context, req *venterprisev1.GetDirectoryProviderUserSummaryRequest) (*venterprisev1.GetDirectoryProviderUserSummaryResponse, error) {

	ret := &venterprisev1.GetDirectoryProviderUserSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindDirectoryProviderUser))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.directoryProviderRef.uid')) AS count_directory_provider`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber, &ret.TotalDirectoryProvider, &ret.TotalUser)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseDirectoryProviderGroup(ctx context.Context, req *venterprisev1.GetDirectoryProviderGroupSummaryRequest) (*venterprisev1.GetDirectoryProviderGroupSummaryResponse, error) {

	ret := &venterprisev1.GetDirectoryProviderGroupSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindDirectoryProviderGroup))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.directoryProviderRef.uid')) AS count_directory_provider`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.groupRef.uid')) AS count_group`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber, &ret.TotalDirectoryProvider, &ret.TotalGroup)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryEnterpriseDeviceManager(ctx context.Context, req *venterprisev1.GetDeviceManagerSummaryRequest) (*venterprisev1.GetDeviceManagerSummaryResponse, error) {

	ret := &venterprisev1.GetDeviceManagerSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(uenterprisev1.KindDeviceManager))
		filters = append(filters, goqu.L(`api`).Eq(uenterprisev1.API))
		filters = append(filters, goqu.L(`version`).Eq(uenterprisev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'CROWDSTRIKE') AS count_crowdstrike`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'SENTINELONE') AS count_sentinelone`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'MICROSOFT_INTUNE') AS count_microsoft_intune`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'JAMF_PRO') AS count_jamf_pro`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'ONEPASSWORD') AS count_onepassword`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'FLEETDM') AS count_fleetdm`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'HUNTRESS') AS count_huntress`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'IRU') AS count_iru`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'OK') AS count_ok`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'LOADING') AS count_loading`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'ERROR') AS count_error`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.state' = 'DEGRADED') AS count_degraded`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.polling.isDisabled') = true) AS count_polling_disabled`),
			goqu.L(`COALESCE(SUM(TRY_CAST(rsc->>'$.status.collection.managedDevices' AS BIGINT)), 0) AS sum_managed_devices`),
			goqu.L(`COALESCE(SUM(TRY_CAST(rsc->>'$.status.linking.linkedDevices' AS BIGINT)), 0) AS sum_linked_devices`),
			goqu.L(`COALESCE(SUM(TRY_CAST(rsc->>'$.status.linking.waitingApproval' AS BIGINT)), 0) AS sum_waiting_approval`),
			goqu.L(`COALESCE(SUM(TRY_CAST(rsc->>'$.status.linking.ambiguous' AS BIGINT)), 0) AS sum_ambiguous`),
			goqu.L(`COALESCE(SUM(TRY_CAST(rsc->>'$.status.linking.failedUpdates' AS BIGINT)), 0) AS sum_failed_updates`),
		)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ret, nil
		}

		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalCrowdStrike, &ret.TotalSentinelOne, &ret.TotalMicrosoftIntune, &ret.TotalJamfPro,
			&ret.TotalOnePassword, &ret.TotalFleetDM, &ret.TotalHuntress, &ret.TotalIru,
			&ret.TotalOK, &ret.TotalLoading, &ret.TotalError, &ret.TotalDegraded,
			&ret.TotalPollingDisabled,
			&ret.TotalManagedDevices, &ret.TotalLinkedDevices, &ret.TotalWaitingApproval,
			&ret.TotalAmbiguous, &ret.TotalFailedUpdates)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}
