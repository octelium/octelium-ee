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

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/main/visibilityv1/vmetav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func getSummaryFilters(api, version, kind string,
	common *vmetav1.CommonSummaryOptions) ([]exp.Expression, error) {

	if err := validateCommonSummaryOptions(common); err != nil {
		return nil, err
	}

	filters := []exp.Expression{
		goqu.L(`kind`).Eq(kind),
		goqu.L(`api`).Eq(api),
		goqu.L(`version`).Eq(version),
		goqu.L(colIsSystemHidden).IsNotTrue(),
	}

	if common.GetFrom() != nil {
		filters = append(filters, goqu.L(colCreatedAt).Gte(common.GetFrom().AsTime().UTC()))
	}

	if common.GetTo() != nil {
		filters = append(filters, goqu.L(colCreatedAt).Lte(common.GetTo().AsTime().UTC()))
	}

	return filters, nil
}

func validateCommonSummaryOptions(opts *vmetav1.CommonSummaryOptions) error {
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

func getSummaryComparisonOptions(common *vmetav1.CommonSummaryOptions) *vmetav1.CommonSummaryOptions {
	if common.GetCompareFrom() == nil && common.GetCompareTo() == nil {
		return nil
	}

	return &vmetav1.CommonSummaryOptions{
		From: common.GetCompareFrom(),
		To:   common.GetCompareTo(),
	}
}

func setSummaryPrevious(cur, prev proto.Message) {
	msg := cur.ProtoReflect()
	fd := msg.Descriptor().Fields().ByName("previous")
	if fd == nil {
		return
	}

	msg.Set(fd, protoreflect.ValueOfMessage(prev.ProtoReflect()))
}

func withSummaryComparison[T proto.Message](ctx context.Context,
	common *vmetav1.CommonSummaryOptions,
	fn func(context.Context, *vmetav1.CommonSummaryOptions) (T, error)) (T, error) {

	ret, err := fn(ctx, common)
	if err != nil {
		return ret, err
	}

	compare := getSummaryComparisonOptions(common)
	if compare == nil {
		return ret, nil
	}

	prev, err := fn(ctx, compare)
	if err != nil {
		return ret, err
	}

	setSummaryPrevious(ret, prev)

	return ret, nil
}
