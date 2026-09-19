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
	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcorev1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
)

func (s *Server) getSummaryCoreUser(ctx context.Context, req *vcorev1.GetUserSummaryRequest) (*vcorev1.GetUserSummaryResponse, error) {

	ret := &vcorev1.GetUserSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindUser, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'HUMAN') AS count_human`, colSpecType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'WORKLOAD') AS count_workload`, colSpecType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_deactivated`, colSpecIsDisabled)),
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
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil

}

func (s *Server) getSummaryCoreSession(ctx context.Context, req *vcorev1.GetSessionSummaryRequest) (*vcorev1.GetSessionSummaryResponse, error) {

	ret := &vcorev1.GetSessionSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindSession, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'CLIENT') AS count_client`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'CLIENTLESS') AS count_clientless`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_connected`, colStatusIsConnected)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_device`, colDeviceUID)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_browser`, colStatusIsBrowser)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OAUTH2') AS count_oauth2`, colStatusCredentialType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ACTIVE') AS count_active`, colSpecState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REJECTED') AS count_rejected`, colSpecState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PENDING') AS count_pending`, colSpecState)),
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
			&ret.TotalClient, &ret.TotalClientless, &ret.TotalConnected,
			&ret.TotalUser, &ret.TotalDevice,
			&ret.TotalClientlessBrowser, &ret.TotalClientlessOAuth2,
			&ret.TotalActive, &ret.TotalRejected, &ret.TotalPending)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryCoreService(ctx context.Context, req *vcorev1.GetServiceSummaryRequest) (*vcorev1.GetServiceSummaryResponse, error) {

	ret := &vcorev1.GetServiceSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindService, req.GetCommon())
	if err != nil {
		return nil, err
	}

	getModeCount := func(mode corev1.Service_Spec_Mode) exp.LiteralExpression {
		return goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = '%s') AS count_%s`,
			colSpecMode, mode.String(), strings.ToLower(mode.String())))
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
			getModeCount(corev1.Service_Spec_SOCKS5),
			getModeCount(corev1.Service_Spec_RDP_WEB),
			getModeCount(corev1.Service_Spec_MCP),
			getModeCount(corev1.Service_Spec_LLM),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_public`, colSpecIsPublic)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_anonymous`, colSpecIsAnonymous)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
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
			&ret.TotalTCP, &ret.TotalUDP, &ret.TotalHTTP,
			&ret.TotalSSH, &ret.TotalKubernetes,
			&ret.TotalPostgres, &ret.TotalMysql,
			&ret.TotalDNS, &ret.TotalGRPC, &ret.TotalWeb,
			&ret.TotalSOCKS5, &ret.TotalRDPWeb,
			&ret.TotalMCP, &ret.TotalLLM,
			&ret.TotalPublic, &ret.TotalAnonymous, &ret.TotalDisabled)
		if err != nil {
			return nil, err
		}

	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil

}

func (s *Server) getSummaryCorePolicy(ctx context.Context, req *vcorev1.GetPolicySummaryRequest) (*vcorev1.GetPolicySummaryResponse, error) {

	ret := &vcorev1.GetPolicySummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindPolicy, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.spec.rules')), 0) AS count_rules`),
			goqu.L(`COALESCE(SUM(
			len(list_filter(
				CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
				x -> json_extract_string(x, '$.effect') = 'ALLOW'
			))
		), 0) AS count_rules_allowed`),
			goqu.L(`COALESCE(SUM(
			len(list_filter(
				CAST(json_extract(rsc, '$.spec.rules') AS JSON[]),
				x -> json_extract_string(x, '$.effect') = 'DENY'
			))
		), 0) AS count_rules_denied`),
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

		err := rows.Scan(&ret.TotalNumber, &ret.TotalDisabled, &ret.TotalRule,
			&ret.TotalRuleAllow, &ret.TotalRuleDenied)
		if err != nil {
			return nil, err
		}

	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryCoreCredential(ctx context.Context, req *vcorev1.GetCredentialSummaryRequest) (*vcorev1.GetCredentialSummaryResponse, error) {

	ret := &vcorev1.GetCredentialSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindCredential, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),

			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'AUTH_TOKEN') AS count_auth_token`, colSpecType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OAUTH2') AS count_oauth2`, colSpecType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ACCESS_TOKEN') AS count_access_token`, colSpecType)),
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
			&ret.TotalDisabled, &ret.TotalUser,
			&ret.TotalAuthenticationToken, &ret.TotalOAuth2, &ret.TotalAccessToken)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil

}

func (s *Server) getSummaryCoreIdentityProvider(ctx context.Context, req *vcorev1.GetIdentityProviderSummaryRequest) (*vcorev1.GetIdentityProviderSummaryResponse, error) {

	ret := &vcorev1.GetIdentityProviderSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindIdentityProvider, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_disabled`, colSpecIsDisabled)),

			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'GITHUB') AS count_github`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OIDC') AS count_oidc`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SAML') AS count_saml`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OIDC_IDENTITY_TOKEN') AS count_oidc_idtoken`, colStatusType)),
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
			&ret.TotalGithub, &ret.TotalOIDC, &ret.TotalSAML, &ret.TotalOIDCIdentityToken)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil

}

func (s *Server) getSummaryCoreDevice(ctx context.Context, req *vcorev1.GetDeviceSummaryRequest) (*vcorev1.GetDeviceSummaryResponse, error) {

	ret := &vcorev1.GetDeviceSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindDevice, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),

			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),

			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ACTIVE') AS count_active`, colSpecState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REJECTED') AS count_rejected`, colSpecState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PENDING') AS count_pending`, colSpecState)),

			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'LINUX') AS count_linux`, colStatusOsType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'WINDOWS') AS count_windows`, colStatusOsType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'MAC') AS count_mac`, colStatusOsType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ANDROID') AS count_android`, colStatusOsType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'IOS') AS count_ios`, colStatusOsType)),
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
			&ret.TotalUser,
			&ret.TotalActive, &ret.TotalRejected, &ret.TotalPending,
			&ret.TotalLinux, &ret.TotalWindows, &ret.TotalMac, &ret.TotalAndroid, &ret.TotalIOS)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryCoreAuthenticator(ctx context.Context, req *vcorev1.GetAuthenticatorSummaryRequest) (*vcorev1.GetAuthenticatorSummaryResponse, error) {

	ret := &vcorev1.GetAuthenticatorSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindAuthenticator, req.GetCommon())
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'TPM') AS count_tpm`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'FIDO') AS count_fido`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'TOTP') AS count_totp`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_device`, colDeviceUID)),

			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ACTIVE') AS count_active`, colSpecState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'REJECTED') AS count_rejected`, colSpecState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PENDING') AS count_pending`, colSpecState)),

			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PLATFORM') AS count_fido_platform`, colStatusFidoType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ROAMING') AS count_fido_roaming`, colStatusFidoType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_fido_passkey`, colStatusFidoIsPasskey)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_fido_hardware`, colStatusFidoIsHardware)),
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
			&ret.TotalTPM,
			&ret.TotalFIDO, &ret.TotalTOTP,
			&ret.TotalUser, &ret.TotalDevice,
			&ret.TotalActive, &ret.TotalRejected, &ret.TotalPending,
			&ret.TotalFIDOPlatform, &ret.TotalFIDORoaming, &ret.TotalFIDOIsPasskey, &ret.TotalFIDOIsHardware)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil

}

func (s *Server) getSummaryCoreGroup(ctx context.Context, req *vcorev1.GetGroupSummaryRequest) (*vcorev1.GetGroupSummaryResponse, error) {

	ret := &vcorev1.GetGroupSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindGroup, req.GetCommon())
	if err != nil {
		return nil, err
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

func (s *Server) getSummaryCoreRegion(ctx context.Context, req *vcorev1.GetRegionSummaryRequest) (*vcorev1.GetRegionSummaryResponse, error) {

	ret := &vcorev1.GetRegionSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindRegion, req.GetCommon())
	if err != nil {
		return nil, err
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

func (s *Server) getSummaryCoreGateway(ctx context.Context, req *vcorev1.GetGatewaySummaryRequest) (*vcorev1.GetGatewaySummaryResponse, error) {

	ret := &vcorev1.GetGatewaySummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindGateway, req.GetCommon())
	if err != nil {
		return nil, err
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

func (s *Server) getSummaryCoreSecret(ctx context.Context, req *vcorev1.GetSecretSummaryRequest) (*vcorev1.GetSecretSummaryResponse, error) {

	ret := &vcorev1.GetSecretSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindSecret, req.GetCommon())
	if err != nil {
		return nil, err
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

func (s *Server) getSummaryCoreNamespace(ctx context.Context, req *vcorev1.GetNamespaceSummaryRequest) (*vcorev1.GetNamespaceSummaryResponse, error) {

	ret := &vcorev1.GetNamespaceSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindNamespace, req.GetCommon())
	if err != nil {
		return nil, err
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
