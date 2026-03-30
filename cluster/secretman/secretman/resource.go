// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package secretman

import (
	"context"
	"encoding/json"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/octelium/octelium/apis/cluster/csecretmanv1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/grpcutils"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/cluster/rscserver/rscserver/rerr"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"go.uber.org/zap"
)

const resourceTableName = "octelium_resources"

func (s *server) setStaleSecretResources(ctx context.Context) error {
	var filters []exp.Expression

	filters = append(filters, goqu.C("kind").Like("%Secret"))
	filters = append(filters, goqu.L(`jsonb_path_exists(resource, '$.data ? (@ != null)')`))

	ds := goqu.From(resourceTableName).Where(filters...).
		Select("resource", goqu.L(`count(*) OVER() AS full_count`))

	ds = ds.OrderAppend(goqu.I(`created_at`).Desc())

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return grpcutils.InternalWithErr(err)
	}

	rows, err := s.db.QueryContext(ctx, sqln, sqlargs...)
	if err != nil {
		return rerr.InternalWithErr(err)
	}

	for rows.Next() {
		var data []byte
		var count int
		if err := rows.Scan(&data, &count); err != nil {
			return rerr.InternalWithErr(err)
		}

		if err := s.checkAndUpdateSecretResource(ctx, data); err != nil {
			zap.L().Error("Could not checkAndUpdateSecretResource", zap.Error(err))
		}
	}

	if err := rows.Err(); err != nil {
		return rerr.InternalWithErr(err)
	}

	return nil
}

func (s *server) checkAndUpdateSecretResource(ctx context.Context, resourceJSON []byte) error {
	rscMap := make(map[string]any)

	if err := json.Unmarshal(resourceJSON, &rscMap); err != nil {
		return err
	}

	if rscMap["data"] == nil {
		return nil
	}

	apiVersion := rscMap["apiVersion"].(string)
	kind := rscMap["kind"].(string)
	mdMap := rscMap["metadata"].(map[string]any)
	md := &metav1.Metadata{}

	if err := pbutils.UnmarshalFromMap(mdMap, md); err != nil {
		return err
	}

	dataJSON, err := json.Marshal(rscMap["data"])
	if err != nil {
		return err
	}

	secretRef := &metav1.ObjectReference{
		ApiVersion:      apiVersion,
		Kind:            kind,
		Uid:             md.Uid,
		Name:            md.Name,
		ResourceVersion: md.ResourceVersion,
	}

	zap.L().Debug("Setting SecretRef", zap.Any("secretRef", secretRef))

	if err := s.doSetDataSecret(ctx, &csecretmanv1.SetSecretRequest{
		Data:      dataJSON,
		SecretRef: secretRef,
	}); err != nil {
		return err
	}

	rscMap["data"] = nil
	outRscJSON, err := json.Marshal(rscMap)
	if err != nil {
		return err
	}

	api, version := vutils.SplitApiVersion(apiVersion)

	return s.doUpdateSecretResource(ctx, outRscJSON, md, api, version, kind)
}

func (s *server) doUpdateSecretResource(ctx context.Context, reqJSONBytes []byte, md *metav1.Metadata, api, version, kind string) error {

	ds := goqu.Update(resourceTableName).Where(goqu.C("uid").Eq(md.Uid)).Set(
		goqu.Record{"resource": string(reqJSONBytes)},
	)

	sqln, sqlargs, err := ds.ToSQL()
	if err != nil {
		return rerr.InternalWithErr(err)
	}

	if _, err := s.db.ExecContext(ctx, sqln, sqlargs...); err != nil {
		return rerr.InternalWithErr(err)
	}

	return nil
}
