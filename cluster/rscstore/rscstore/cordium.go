package rscstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/pkg/apiutils/ucordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vcordiumv1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
)

func getCordiumWorkspaceActiveExpr() string {
	return fmt.Sprintf(`(%s IS NOT NULL) AND (%s != 'STOPPED')`, colStatusState, colStatusState)
}

func getCordiumBuildCountExpr(state string) string {
	return fmt.Sprintf(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.status.buildInfo.builds') AS JSON[]),
					x -> json_extract_string(x, '$.state') = '%s'
				))
			), 0)`, state)
}

func (s *Server) doSummaryCordiumWorkspace(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetWorkspaceSummaryResponse, error) {

	ret := &vcordiumv1.GetWorkspaceSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindWorkspace, common)
	if err != nil {
		return nil, err
	}

	activeExpr := getCordiumWorkspaceActiveExpr()

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s IN ('INIT_REQUEST', 'INITIALIZING')) AS count_initializing`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PULLING_IMAGE') AS count_pulling_image`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'BUILDING_IMAGE') AS count_building_image`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'STARTING_RUNTIME') AS count_starting_runtime`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'PREPARING') AS count_preparing`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'RUNNING') AS count_running`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s IN ('STOPPING_REQUEST', 'STOPPING')) AS count_stopping`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'STOPPED') AS count_stopped`, colStatusState)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'failure')) AS count_failed`, colStatusRunKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_build`, colStatusIsBuild)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_ephemeral`, colSpecIsEphemeral)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_auto_stop`, colSpecAutoStop)),
			goqu.L(`COUNT(*) FILTER (WHERE json_array_length(rsc, '$.status.sharedPorts') > 0) AS count_shared`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'MEMBERS')) AS count_shared_members`, colStatusSharedPortModes)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'ALL')) AS count_shared_all`, colStatusSharedPortModes)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'USER') AS count_user_space`, colStatusSpaceType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ORGANIZATION') AS count_organization_space`, colStatusSpaceType)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_session`, colSessionUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_space`, colSpaceUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_template`, colTemplateUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_region`, colRegionUID)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)) FILTER (WHERE %s), 0) AS sum_cpu_millicores`, colStatusCPUMillicores, activeExpr)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)) FILTER (WHERE %s), 0) AS sum_memory_megabytes`, colStatusMemoryMegabytes, activeExpr)),
			goqu.L(fmt.Sprintf(`COALESCE(SUM(TRY_CAST(%s AS BIGINT)) FILTER (WHERE %s), 0) AS sum_storage_megabytes`, colStatusStorageMegabytes, activeExpr)),
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
			&ret.TotalInitializing, &ret.TotalPullingImage, &ret.TotalBuildingImage,
			&ret.TotalStartingRuntime, &ret.TotalPreparing, &ret.TotalRunning,
			&ret.TotalStopping, &ret.TotalStopped, &ret.TotalFailed,
			&ret.TotalBuild, &ret.TotalEphemeral, &ret.TotalAutoStop,
			&ret.TotalShared, &ret.TotalSharedMembers, &ret.TotalSharedAll,
			&ret.TotalUserSpace, &ret.TotalOrganizationSpace,
			&ret.TotalUser, &ret.TotalSession, &ret.TotalSpace, &ret.TotalTemplate, &ret.TotalRegion,
			&ret.TotalCPUMillicores, &ret.TotalMemoryMegabytes, &ret.TotalStorageMegabytes)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumTemplate(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetTemplateSummaryResponse, error) {

	ret := &vcordiumv1.GetTemplateSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindTemplate, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(`COALESCE(SUM(json_array_length(rsc, '$.status.buildInfo.builds')), 0) AS count_build`),
			goqu.L(fmt.Sprintf(`%s AS count_build_ready`, getCordiumBuildCountExpr("STATE_READY"))),
			goqu.L(fmt.Sprintf(`%s AS count_build_running`, getCordiumBuildCountExpr("STATE_RUNNING"))),
			goqu.L(fmt.Sprintf(`%s AS count_build_failed`, getCordiumBuildCountExpr("STATE_FAILED"))),
			goqu.L(`COALESCE(SUM(
				len(list_filter(
					CAST(json_extract(rsc, '$.status.buildInfo.builds') AS JSON[]),
					x -> json_extract_string(x, '$.isCanceled') = 'true'
				))
			), 0) AS count_build_canceled`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE COALESCE(%s, '') != '') AS count_with_ready_build`, colStatusReadyBuildID)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE COALESCE(%s, '') != '') AS count_with_running_build`, colStatusRunningBuildID)),
			goqu.L(`COUNT(*) FILTER (WHERE COALESCE(json_array_length(rsc, '$.status.buildInfo.builds'), 0) = 0) AS count_never_built`),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_space`, colSpaceUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_git_provider`, colGitProviderUID)),
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
			&ret.TotalBuild, &ret.TotalBuildReady, &ret.TotalBuildRunning,
			&ret.TotalBuildFailed, &ret.TotalBuildCanceled,
			&ret.TotalWithReadyBuild, &ret.TotalWithRunningBuild, &ret.TotalNeverBuilt,
			&ret.TotalUser, &ret.TotalSpace, &ret.TotalGitProvider)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumSpace(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetSpaceSummaryResponse, error) {

	ret := &vcordiumv1.GetSpaceSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindSpace, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'USER') AS count_user_space`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ORGANIZATION') AS count_organization_space`, colStatusType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_ssh_disabled`, colSpecDisableSSH)),
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
		err := rows.Scan(&ret.TotalNumber,
			&ret.TotalUserSpace, &ret.TotalOrganizationSpace, &ret.TotalSSHDisabled,
			&ret.TotalUser)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	ret.TotalMembership, err = s.countCordiumResourcesBySpace(ctx, ucordiumv1.KindMembership, filters)
	if err != nil {
		return nil, err
	}

	ret.TotalWorkspace, err = s.countCordiumResourcesBySpace(ctx, ucordiumv1.KindWorkspace, filters)
	if err != nil {
		return nil, err
	}

	ret.TotalTemplate, err = s.countCordiumResourcesBySpace(ctx, ucordiumv1.KindTemplate, filters)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (s *Server) countCordiumResourcesBySpace(ctx context.Context, kind string, spaceFilters []exp.Expression) (uint64, error) {

	spaceDS := goqu.From("resources").Where(spaceFilters...).Select(goqu.L(`uid`))

	var filters []exp.Expression
	{
		filters = append(filters, goqu.L(`kind`).Eq(kind))
		filters = append(filters, goqu.L(`api`).Eq(ucordiumv1.API))
		filters = append(filters, goqu.L(`version`).Eq(ucordiumv1.Version))
		filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())
		filters = append(filters, goqu.L(colSpaceUID).In(spaceDS))
	}

	ds := goqu.From("resources").Where(filters...).Select(goqu.L(`COUNT(*) AS count_total`))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return 0, grpcutils.InternalWithErr(err)
	}

	var ret uint64
	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&ret); err != nil {
		return 0, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumMembership(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetMembershipSummaryResponse, error) {

	ret := &vcordiumv1.GetMembershipSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindMembership, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'OWNER') AS count_role_owner`, colSpecRole)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'ADMIN') AS count_role_admin`, colSpecRole)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'USER') AS count_role_user`, colSpecRole)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_space`, colSpaceUID)),
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
			&ret.TotalRoleOwner, &ret.TotalRoleAdmin, &ret.TotalRoleUser,
			&ret.TotalUser, &ret.TotalSpace)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumGitProvider(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetGitProviderSummaryResponse, error) {

	ret := &vcordiumv1.GetGitProviderSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindGitProvider, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'github')) AS count_github`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'gitlab')) AS count_gitlab`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE list_contains(%s, 'oauth2')) AS count_oauth2`, colSpecKeys)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_space`, colSpaceUID)),
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
			&ret.TotalGithub, &ret.TotalGitlab, &ret.TotalOAuth2,
			&ret.TotalUser, &ret.TotalSpace)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumSecret(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetSecretSummaryResponse, error) {

	ret := &vcordiumv1.GetSecretSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindSecret, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_user`, colUserUID)),
			goqu.L(fmt.Sprintf(`COUNT(DISTINCT %s) AS count_space`, colSpaceUID)),
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
		err := rows.Scan(&ret.TotalNumber, &ret.TotalUser, &ret.TotalSpace)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumUserSecret(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetUserSecretSummaryResponse, error) {

	ret := &vcordiumv1.GetUserSecretSummaryResponse{}
	filters, err := getSummaryFilters(ucordiumv1.API, ucordiumv1.Version, ucordiumv1.KindUserSecret, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE (%s IS NULL) OR (%s = 'DEFAULT')) AS count_default`, colSpecType, colSpecType)),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = 'SSH_KEY') AS count_ssh_key`, colSpecType)),
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
		err := rows.Scan(&ret.TotalNumber, &ret.TotalDefault, &ret.TotalSSHKey, &ret.TotalUser)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) doSummaryCordiumRegion(ctx context.Context, common *vmetav1.CommonSummaryOptions) (*vcordiumv1.GetRegionSummaryResponse, error) {

	ret := &vcordiumv1.GetRegionSummaryResponse{}
	filters, err := getSummaryFilters(ucorev1.API, ucorev1.Version, ucorev1.KindRegion, common)
	if err != nil {
		return nil, err
	}

	ds := goqu.From("resources").Where(filters...).
		Select(
			goqu.L(`COUNT(*) AS count_total`),
			goqu.L(fmt.Sprintf(`COUNT(*) FILTER (WHERE %s = true) AS count_enabled`, colStatusExtCordiumIsEnabled)),
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
		err := rows.Scan(&ret.TotalNumber, &ret.TotalEnabled)
		if err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret, nil
}

func (s *Server) getSummaryCordiumWorkspace(ctx context.Context, req *vcordiumv1.GetWorkspaceSummaryRequest) (*vcordiumv1.GetWorkspaceSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumWorkspace)
}

func (s *Server) getSummaryCordiumTemplate(ctx context.Context, req *vcordiumv1.GetTemplateSummaryRequest) (*vcordiumv1.GetTemplateSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumTemplate)
}

func (s *Server) getSummaryCordiumSpace(ctx context.Context, req *vcordiumv1.GetSpaceSummaryRequest) (*vcordiumv1.GetSpaceSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumSpace)
}

func (s *Server) getSummaryCordiumMembership(ctx context.Context, req *vcordiumv1.GetMembershipSummaryRequest) (*vcordiumv1.GetMembershipSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumMembership)
}

func (s *Server) getSummaryCordiumGitProvider(ctx context.Context, req *vcordiumv1.GetGitProviderSummaryRequest) (*vcordiumv1.GetGitProviderSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumGitProvider)
}

func (s *Server) getSummaryCordiumSecret(ctx context.Context, req *vcordiumv1.GetSecretSummaryRequest) (*vcordiumv1.GetSecretSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumSecret)
}

func (s *Server) getSummaryCordiumUserSecret(ctx context.Context, req *vcordiumv1.GetUserSecretSummaryRequest) (*vcordiumv1.GetUserSecretSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumUserSecret)
}

func (s *Server) getSummaryCordiumRegion(ctx context.Context, req *vcordiumv1.GetRegionSummaryRequest) (*vcordiumv1.GetRegionSummaryResponse, error) {
	return withSummaryComparison(ctx, req.GetCommon(), s.doSummaryCordiumRegion)
}
