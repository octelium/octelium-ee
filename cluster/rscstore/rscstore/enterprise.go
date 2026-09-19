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
	"fmt"
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'otlp')) AS count_otlp`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'otlpHTTP')) AS count_otlp_http`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'clickhouse')) AS count_clickhouse`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'elasticsearch')) AS count_elasticsearch`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'logzio')) AS count_logzio`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'influxDB')) AS count_influxdb`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'kafka')) AS count_kafka`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'datadog')) AS count_datadog`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'splunk')) AS count_splunk`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'azureMonitor')) AS count_azure_monitor`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'azureDataExplorer')) AS count_azure_data_explorer`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'prometheusRemoteWrite')) AS count_prometheus_remote_write`, colSpecKeys)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'TYPE_AZURE_KEY_VAULT') AS count_azure_key_vault`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'TYPE_HASHICORP_VAULT') AS count_hashicorp_vault`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'TYPE_GCP_KMS') AS count_gcp_kms`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'TYPE_AWS_KMS') AS count_aws_kms`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'KUBERNETES') AS count_kubernetes`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OK') AS count_ok`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'LOADING') AS count_loading`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SYNCING') AS count_synchronizing`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUCCESS') AS count_sync_success`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FAILED') AS count_sync_failed`, colStatusSyncState)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	now := time.Now().UTC()
	expiringSoon := now.Add(30 * 24 * time.Hour)

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'MANAGED') AS count_managed`, colSpecMode)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'MANUAL') AS count_manual`, colSpecMode)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ISSUANCE_REQUESTED') AS count_issuance_requested`, colStatusIssuanceState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ISSUING') AS count_issuing`, colStatusIssuanceState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUCCESS') AS count_issuance_success`, colStatusIssuanceState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FAILED') AS count_issuance_failed`, colStatusIssuanceState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s < ?) AS count_expired`, colStatusIssuanceExpiry), now),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE (%s >= ?) AND (%s < ?)) AS count_expiring_soon`, colStatusIssuanceExpiry, colStatusIssuanceExpiry),
				now, expiringSoon),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'serviceRef')) AS count_service`, colStatusKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'namespaceRef')) AS count_namespace`, colStatusKeys)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_certificate_issuer`, colCertificateIssuerUID)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'acme')) AS count_acme`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PREPARING') AS count_preparing`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'READY') AS count_ready`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'NOT_READY') AS count_not_ready`, colStatusState)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'cloudflare')) AS count_cloudflare`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'aws')) AS count_aws`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'digitalocean')) AS count_digitalocean`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'google')) AS count_google`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'azure')) AS count_azure`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'linode')) AS count_linode`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'ovh')) AS count_ovh`, colSpecKeys)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'scim')) AS count_scim`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'googleWorkspace')) AS count_google_workspace`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'keycloak')) AS count_keycloak`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SYNCING') AS count_synchronizing`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SUCCESS') AS count_sync_success`, colStatusSyncState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FAILED') AS count_sync_failed`, colStatusSyncState)),
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
		userFilters = append(userFilters, goqu.L(colIsSystemHidden).IsNotTrue())
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
		groupFilters = append(groupFilters, goqu.L(colIsSystemHidden).IsNotTrue())
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_directory_provider`, colDirectoryProviderUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_directory_provider`, colDirectoryProviderUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_group`, colGroupUID)),
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
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'CROWDSTRIKE') AS count_crowdstrike`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SENTINELONE') AS count_sentinelone`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'MICROSOFT_INTUNE') AS count_microsoft_intune`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'JAMF_PRO') AS count_jamf_pro`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ONEPASSWORD') AS count_onepassword`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FLEETDM') AS count_fleetdm`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'HUNTRESS') AS count_huntress`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'IRU') AS count_iru`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OK') AS count_ok`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'LOADING') AS count_loading`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ERROR') AS count_error`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'DEGRADED') AS count_degraded`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_polling_disabled`, colSpecPollingIsDisabled)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)), 0) AS sum_managed_devices`, colStatusManagedDevices)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)), 0) AS sum_linked_devices`, colStatusLinkedDevices)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)), 0) AS sum_waiting_approval`, colStatusWaitingApprove)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)), 0) AS sum_ambiguous`, colStatusAmbiguous)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)), 0) AS sum_failed_updates`, colStatusFailedUpdates)),
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
