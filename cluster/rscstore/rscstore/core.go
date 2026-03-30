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
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
)

func (s *Server) getSummaryCoreUser(ctx context.Context, req *vcorev1.GetUserSummaryRequest) (*vcorev1.GetUserSummaryResponse, error) {

	ret := &vcorev1.GetUserSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindUser))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.type' = 'HUMAN') AS count_human`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.type' = 'WORKLOAD') AS count_workload`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_deactivated`),
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

	for rows.Next() {
		var countHuman int
		var countWorkload int
		var countTotal int
		var countDeactivated int

		err := rows.Scan(&countTotal, &countHuman, &countWorkload, &countDeactivated)
		if err != nil {
			return nil, err
		}

		ret.TotalNumber = uint32(countTotal)
		ret.TotalHuman = uint32(countHuman)
		ret.TotalWorkload = uint32(countWorkload)
		ret.TotalDisabled = uint32(countDeactivated)
	}

	return ret, nil

}

func (s *Server) getSummaryCoreSession(ctx context.Context, req *vcorev1.GetSessionSummaryRequest) (*vcorev1.GetSessionSummaryResponse, error) {

	ret := &vcorev1.GetSessionSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindSession))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'CLIENT') AS count_client`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'CLIENTLESS') AS count_clientless`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.status.isConnected') = true) AS count_connected`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.deviceRef.uid')) AS count_device`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.clientlessType' = 'BROWSER') AS count_browser`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.clientlessType' = 'SDK') AS count_sdk`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.clientlessType' = 'OAUTH2') AS count_oauth2`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'ACTIVE') AS count_active`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'REJECTED') AS count_rejected`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'PENDING') AS count_pending`),
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalClient, &ret.TotalClientless, &ret.TotalConnected,
			&ret.TotalUser, &ret.TotalDevice,
			&ret.TotalClientlessBrowser, &ret.TotalClientlessSDK, &ret.TotalClientlessOAuth2,
			&ret.TotalActive, &ret.TotalRejected, &ret.TotalPending)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *Server) getSummaryCoreService(ctx context.Context, req *vcorev1.GetServiceSummaryRequest) (*vcorev1.GetServiceSummaryResponse, error) {

	ret := &vcorev1.GetServiceSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindService))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	getModeCount := func(mode corev1.Service_Spec_Mode) exp.LiteralExpression {
		return goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE rsc->>'$.spec.mode' = '%s') AS count_%s`,
			mode.String(), strings.ToLower(mode.String())))
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			getModeCount(corev1.Service_Spec_TCP),
			getModeCount(corev1.Service_Spec_UDP),
			getModeCount(corev1.Service_Spec_HTTP),
			getModeCount(corev1.Service_Spec_SSH),
			getModeCount(corev1.Service_Spec_KUBERNETES),
			getModeCount(corev1.Service_Spec_POSTGRES),
			getModeCount(corev1.Service_Spec_MYSQL),
			getModeCount(corev1.Service_Spec_DNS),
			getModeCount(corev1.Service_Spec_GRPC),
			getModeCount(corev1.Service_Spec_WEB),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isPublic') = true) AS count_public`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isAnonymous') = true) AS count_anonymous`),
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

	for rows.Next() {

		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalTCP, &ret.TotalUDP, &ret.TotalHTTP,
			&ret.TotalSSH, &ret.TotalKubernetes,
			&ret.TotalPostgres, &ret.TotalMysql,
			&ret.TotalDNS, &ret.TotalGRPC, &ret.TotalWeb,
			&ret.TotalPublic, &ret.TotalAnonymous)
		if err != nil {
			return nil, err
		}

	}

	return ret, nil

}

func (s *Server) getSummaryCorePolicy(ctx context.Context, req *vcorev1.GetPolicySummaryRequest) (*vcorev1.GetPolicySummaryResponse, error) {

	ret := &vcorev1.GetPolicySummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindPolicy))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_disabled`),
			goqu.L(`COUNT(len(json_extract(rsc, '$.spec.rules'))) AS count_rules`),
			// goqu.L("SUM((rule ->> 'effect') = 'ALLOWED')").As("count_allowed"),
			// goqu.L("SUM((rule ->> 'effect') = 'DENIED')").As("count_denied"),
			goqu.L(`COUNT(len(json_extract(rsc, '$.spec.rules'))) FILTER (WHERE json_extract_string(json_extract(rsc, '$.spec.rules[0]'), '$.effect') = 'ALLOW') AS count_rules_allowed`),
			goqu.L(`COUNT(len(json_extract(rsc, '$.spec.rules'))) FILTER (WHERE json_extract_string(json_extract(rsc, '$.spec.rules[0]'), '$.effect') = 'DENY') AS count_rules_denied`),
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

	for rows.Next() {

		err := rows.Scan(&ret.TotalNumber, &ret.TotalDisabled, &ret.TotalRule,
			&ret.TotalRuleAllow, &ret.TotalRuleDenied)
		if err != nil {
			return nil, err
		}

	}

	return ret, nil
}

func (s *Server) getSummaryCoreCredential(ctx context.Context, req *vcorev1.GetCredentialSummaryRequest) (*vcorev1.GetCredentialSummaryResponse, error) {

	ret := &vcorev1.GetCredentialSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindCredential))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_disabled`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),

			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.type' = 'AUTH_TOKEN') AS count_auth_token`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.type' = 'OAUTH2') AS count_oauth2`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.type' = 'ACCESS_TOKEN') AS count_access_token`),
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalDisabled, &ret.TotalUser,
			&ret.TotalAuthenticationToken, &ret.TotalOAuth2, &ret.TotalAccessToken)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil

}

func (s *Server) getSummaryCoreIdentityProvider(ctx context.Context, req *vcorev1.GetIdentityProviderSummaryRequest) (*vcorev1.GetIdentityProviderSummaryResponse, error) {

	ret := &vcorev1.GetIdentityProviderSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindIdentityProvider))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.spec.isDisabled') = true) AS count_disabled`),

			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'GITHUB') AS count_github`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'OIDC') AS count_oidc`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'SAML') AS count_saml`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'OIDC_IDENTITY_TOKEN') AS count_oidc_idtoken`),
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalDisabled,
			&ret.TotalGithub, &ret.TotalOIDC, &ret.TotalSAML, &ret.TotalOIDCIdentityToken)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil

}

func (s *Server) getSummaryCoreDevice(ctx context.Context, req *vcorev1.GetDeviceSummaryRequest) (*vcorev1.GetDeviceSummaryResponse, error) {

	ret := &vcorev1.GetDeviceSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindDevice))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),

			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),

			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'ACTIVE') AS count_active`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'REJECTED') AS count_rejected`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'PENDING') AS count_pending`),

			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.osType' = 'LINUX') AS count_linux`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.osType' = 'WINDOWS') AS count_windows`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.osType' = 'MAC') AS count_mac`),
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalUser,
			&ret.TotalActive, &ret.TotalRejected, &ret.TotalPending,
			&ret.TotalLinux, &ret.TotalWindows, &ret.TotalMac)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *Server) getSummaryCoreAuthenticator(ctx context.Context, req *vcorev1.GetAuthenticatorSummaryRequest) (*vcorev1.GetAuthenticatorSummaryResponse, error) {

	ret := &vcorev1.GetAuthenticatorSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindAuthenticator))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
		filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'TPM') AS count_tpm`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'FIDO') AS count_fido`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.type' = 'TOTP') AS count_totp`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.userRef.uid')) AS count_user`),
			goqu.L(`COUNT(DISTINCT json_extract_string(rsc, '$.status.deviceRef.uid')) AS count_device`),

			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'ACTIVE') AS count_active`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'REJECTED') AS count_rejected`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.spec.state' = 'PENDING') AS count_pending`),

			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.info.fido.type' = 'PLATFORM') AS count_fido_platform`),
			goqu.L(`COUNT(*) FILTER (WHERE rsc->>'$.status.info.fido.type' = 'ROAMING') AS count_fido_roaming`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.status.info.fido.isPasskey') = true) AS count_fido_passkey`),
			goqu.L(`COUNT(*) FILTER (WHERE json_extract(rsc, '$.status.info.fido.isHardware') = true) AS count_fido_hardware`),
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalTPM,
			&ret.TotalFIDO, &ret.TotalTOTP,
			&ret.TotalUser, &ret.TotalDevice,
			&ret.TotalActive, &ret.TotalRejected, &ret.TotalPending,
			&ret.TotalFIDOPlatform, &ret.TotalFIDORoaming, &ret.TotalFIDOIsPasskey, &ret.TotalFIDOIsHardware)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil

}

func (s *Server) getSummaryCoreGroup(ctx context.Context, req *vcorev1.GetGroupSummaryRequest) (*vcorev1.GetGroupSummaryResponse, error) {

	ret := &vcorev1.GetGroupSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindGroup))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *Server) getSummaryCoreRegion(ctx context.Context, req *vcorev1.GetRegionSummaryRequest) (*vcorev1.GetRegionSummaryResponse, error) {

	ret := &vcorev1.GetRegionSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindRegion))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *Server) getSummaryCoreGateway(ctx context.Context, req *vcorev1.GetGatewaySummaryRequest) (*vcorev1.GetGatewaySummaryResponse, error) {

	ret := &vcorev1.GetGatewaySummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindGateway))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *Server) getSummaryCoreSecret(ctx context.Context, req *vcorev1.GetSecretSummaryRequest) (*vcorev1.GetSecretSummaryResponse, error) {

	ret := &vcorev1.GetSecretSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindSecret))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *Server) getSummaryCoreNamespace(ctx context.Context, req *vcorev1.GetNamespaceSummaryRequest) (*vcorev1.GetNamespaceSummaryResponse, error) {

	ret := &vcorev1.GetNamespaceSummaryResponse{}
	var filters []exp.Expression

	{
		filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindNamespace))
		filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
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

	for rows.Next() {
		err := rows.Scan(&ret.TotalNumber)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}
