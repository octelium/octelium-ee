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
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/common/rgx"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type doListReq struct {
	filters []exp.Expression
	api     string
	version string
	kind    string
	common  *vmetav1.CommonListOptions
}

const defaultItemsPerPage = 10
const maxItemsPerPage = 1000

func (s *Server) doList(ctx context.Context, req *doListReq) (proto.Message, error) {
	var filters []exp.Expression

	listMeta := &metav1.ListResponseMeta{}
	if req.common == nil {
		req.common = &vmetav1.CommonListOptions{}
	}

	if err := validateCommonListOptions(req.common); err != nil {
		return nil, err
	}

	query := strings.TrimSpace(req.common.Query)
	hasQuery := query != ""
	if len(query) > 100 {
		return nil, grpcutils.InvalidArg("Query is too long")
	}

	zap.L().Debug("New list req",
		zap.String("api", req.api), zap.String("kind", req.kind), zap.Any("req", req.common))

	filters = append(filters, goqu.L(`api`).Eq(req.api))
	filters = append(filters, goqu.L(`version`).Eq(req.version))
	filters = append(filters, goqu.L(`kind`).Eq(req.kind))
	filters = append(filters, goqu.L(colIsSystemHidden).IsNotTrue())

	if req.common.From != nil {
		filters = append(filters, goqu.L(colCreatedAt).Gte(req.common.From.AsTime().UTC()))
	}

	if req.common.To != nil {
		filters = append(filters, goqu.L(colCreatedAt).Lte(req.common.To.AsTime().UTC()))
	}

	normalizedQuery := strings.ToLower(query)
	if hasQuery {
		filters = append(filters, goqu.L(`contains(rsc_str, ?)`, normalizedQuery))
	}

	filters = append(filters, req.filters...)

	if req.common.Tag != "" {
		if !rgx.NameMain.MatchString(req.common.Tag) {
			return nil, grpcutils.InvalidArg("Invalid tag: %s", req.common.Tag)
		}

		filters = append(filters, goqu.L(fmt.Sprintf(`list_contains(%s, ?)`, colTags), req.common.Tag))
	}

	limit := req.common.ItemsPerPage
	if req.common.Page > 10000 {
		return nil, grpcutils.InvalidArgWithErr(errors.Errorf("Page number is too high"))
	}

	if limit == 0 {
		limit = defaultItemsPerPage
	} else if limit > maxItemsPerPage {
		limit = maxItemsPerPage
	}

	listMeta.ItemsPerPage = limit
	listMeta.Page = req.common.Page

	ds := goqu.From("resources").
		Prepared(true).
		Where(filters...).
		Select(goqu.L(fmt.Sprintf(`CAST(rsc AS %s)`, kindVarchar))).
		Offset(uint(req.common.Page * limit)).
		Limit(uint(limit))

	if hasQuery {
		ds = ds.OrderAppend(getQueryRankExpr(normalizedQuery).Desc())
	}

	ds = ds.OrderAppend(getOrderByExpr(req.common.OrderBy))

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

	for rows.Next() {
		var rscJSON []byte

		if err := rows.Scan(&rscJSON); err != nil {
			return nil, grpcutils.InternalWithErr(err)
		}

		rsc, err := ovutils.NewResourceObject(req.api, req.version, req.kind)
		if err != nil {
			return nil, err
		}

		if err := pbutils.UnmarshalJSON(rscJSON, rsc); err != nil {
			return nil, err
		}

		items = append(items, rsc)
	}

	if err := rows.Err(); err != nil {
		return nil, grpcutils.InternalWithErr(err)
	}

	if len(items) == 0 && listMeta.Page > 0 {
		return nil, grpcutils.NotFound("Not Items found for that page")
	}

	if listMeta.Page == 0 && uint32(len(items)) < limit {
		listMeta.TotalCount = uint32(len(items))
	} else {
		listMeta.TotalCount, err = s.getListTotalCount(ctx, filters)
		if err != nil {
			return nil, err
		}
	}

	if listMeta.TotalCount > (listMeta.Page+1)*listMeta.ItemsPerPage {
		listMeta.HasMore = true
	}

	return s.toResourceList(items, listMeta, req.api, req.version, req.kind)
}

func (s *Server) getListTotalCount(ctx context.Context, filters []exp.Expression) (uint32, error) {
	ds := goqu.From("resources").
		Prepared(true).
		Where(filters...).
		Select(goqu.L(`COUNT(*)`))

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return 0, grpcutils.InternalWithErr(err)
	}

	var count int64
	if err := s.db.QueryRowContext(ctx, sqln, sqlargs...).Scan(&count); err != nil {
		return 0, grpcutils.InternalWithErr(err)
	}

	if count < 0 {
		return 0, nil
	}

	return uint32(count), nil
}

func getQueryRankExpr(normalizedQuery string) exp.LiteralExpression {
	return goqu.L(fmt.Sprintf(`
	CASE
		WHEN lower(%s) = ? THEN 5
		WHEN lower(%s) = ? THEN 4
		WHEN starts_with(lower(%s), ?) OR starts_with(lower(%s), ?) THEN 3
		WHEN contains(lower(%s), ?) OR contains(lower(%s), ?) THEN 2
		ELSE 1
	END`, colName, colDisplayName, colName, colDisplayName, colName, colDisplayName),
		normalizedQuery, normalizedQuery, normalizedQuery,
		normalizedQuery, normalizedQuery, normalizedQuery)
}

func getOrderByExpr(orderBy *vmetav1.CommonListOptions_OrderBy) exp.OrderedExpression {
	if orderBy == nil {
		return goqu.L(colCreatedAt).Desc()
	}

	var column string
	switch orderBy.Type {
	case vmetav1.CommonListOptions_OrderBy_CREATED_AT:
		column = colCreatedAt
	case vmetav1.CommonListOptions_OrderBy_NAME:
		column = colName
	default:
		return goqu.L(colCreatedAt).Desc()
	}

	if orderBy.Mode == vmetav1.CommonListOptions_OrderBy_DESC {
		return goqu.L(column).Desc()
	}

	return goqu.L(column).Asc()
}

func (s *Server) toResourceList(lst []umetav1.ResourceObjectI, listMeta *metav1.ListResponseMeta, api, version, kind string) (proto.Message, error) {
	objList, err := ovutils.NewResourceObjectList(api, version, kind)
	if err != nil {
		return nil, err
	}

	msg := objList.ProtoReflect()
	fields := msg.Descriptor().Fields()

	setListField := func(name string, val protoreflect.Value) error {
		fd := fields.ByName(protoreflect.Name(name))
		if fd == nil {
			return errors.Errorf("The %sList message has no %s field", kind, name)
		}

		msg.Set(fd, val)
		return nil
	}

	if err := setListField("apiVersion",
		protoreflect.ValueOfString(vutils.GetApiVersion(api, version))); err != nil {
		return nil, err
	}

	if err := setListField("kind",
		protoreflect.ValueOfString(fmt.Sprintf("%sList", kind))); err != nil {
		return nil, err
	}

	if listMeta != nil {
		if err := setListField("listResponseMeta",
			protoreflect.ValueOfMessage(listMeta.ProtoReflect())); err != nil {
			return nil, err
		}
	}

	itemsField := fields.ByName("items")
	if itemsField == nil {
		return nil, errors.Errorf("The %sList message has no items field", kind)
	}

	items := msg.Mutable(itemsField).List()
	for _, itm := range lst {
		items.Append(protoreflect.ValueOfMessage(itm.ProtoReflect()))
	}

	return objList, nil
}

func validateCommonListOptions(opts *vmetav1.CommonListOptions) error {
	if opts == nil {
		return nil
	}

	if opts.From != nil {
		if err := opts.From.CheckValid(); err != nil {
			return grpcutils.InvalidArg("Invalid from timestamp: %s", err.Error())
		}
	}

	if opts.To != nil {
		if err := opts.To.CheckValid(); err != nil {
			return grpcutils.InvalidArg("Invalid to timestamp: %s", err.Error())
		}
	}

	if opts.From != nil && opts.To != nil && opts.From.AsTime().After(opts.To.AsTime()) {
		return grpcutils.InvalidArg("The from timestamp must not be after the to timestamp")
	}

	return nil
}
