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
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium-ee/cluster/common/ovutils"
	"github.com/octelium/octelium/apis/cluster/caccessv1"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
)

func (s *Server) listSubjectUser(ctx context.Context, req *caccessv1.ListSubjectUserRequest) (*corev1.UserList, error) {

	query := strings.TrimSpace(req.Query)
	if len(query) > 100 {
		return nil, grpcutils.InvalidArg("Query is too long")
	}

	var filters []exp.Expression

	filters = append(filters, goqu.L(`api`).Eq(ucorev1.API))
	filters = append(filters, goqu.L(`version`).Eq(ucorev1.Version))
	filters = append(filters, goqu.L(`kind`).Eq(ucorev1.KindUser))
	filters = append(filters, goqu.L(`rsc->>'$.metadata.isSystemHidden'`).IsNotTrue())
	filters = append(filters, goqu.L(`rsc->>'$.spec.type'`).Eq(corev1.User_Spec_HUMAN.String()))
	filters = append(filters, goqu.L(`json_extract(rsc, '$.spec.isDisabled')`).IsNotTrue())

	if query != "" {
		pattern := fmt.Sprintf(`%%%s%%`, escapeLikePattern(query))

		filters = append(filters, goqu.Or(
			goqu.L(`json_extract_string(rsc, '$.metadata.name') ILIKE ? ESCAPE '\'`, pattern),
			goqu.L(`json_extract_string(rsc, '$.metadata.displayName') ILIKE ? ESCAPE '\'`, pattern),
			goqu.L(`json_extract_string(rsc, '$.spec.email') ILIKE ? ESCAPE '\'`, pattern),
		))
	}

	limit := req.ItemsPerPage
	if limit == 0 {
		limit = defaultItemsPerPage
	} else if limit > maxItemsPerPage {
		limit = maxItemsPerPage
	}

	ds := goqu.From("resources").
		Prepared(true).
		Where(filters...).
		Select(
			goqu.L(`COUNT(*) OVER() as count`),
			goqu.L(`rsc`),
		).
		Offset(uint(req.Page * limit)).
		Limit(uint(limit)).
		OrderAppend(goqu.L(`rsc->'metadata'->>'name'`).Asc())

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}
	defer rows.Close()

	var items []umetav1.ResourceObjectI
	listMeta := &metav1.ListResponseMeta{
		Page:         req.Page,
		ItemsPerPage: limit,
	}

	for rows.Next() {
		rscMap := make(map[string]any)
		var count int

		if err := rows.Scan(&count, &rscMap); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		listMeta.TotalCount = uint32(count)

		rsc, err := ovutils.NewResourceObject(ucorev1.API, ucorev1.Version, ucorev1.KindUser)
		if err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		if err := pbutils.UnmarshalFromMap(rscMap, rsc); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		items = append(items, rsc)
	}

	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if listMeta.TotalCount > (listMeta.Page+1)*listMeta.ItemsPerPage {
		listMeta.HasMore = true
	}

	ret, err := s.toResourceList(items, listMeta, ucorev1.API, ucorev1.Version, ucorev1.KindUser)
	if err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	return ret.(*corev1.UserList), nil
}

func escapeLikePattern(arg string) string {
	arg = strings.ReplaceAll(arg, `\`, `\\`)
	arg = strings.ReplaceAll(arg, `%`, `\%`)
	arg = strings.ReplaceAll(arg, `_`, `\_`)

	return arg
}
