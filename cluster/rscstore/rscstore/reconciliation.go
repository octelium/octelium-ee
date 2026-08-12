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
	"fmt"
	"reflect"
	"time"

	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type resourceKind struct {
	api     string
	version string
	kind    string
}

func coreResourceKinds() []resourceKind {
	return []resourceKind{
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindUser},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindSession},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindDevice},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindGroup},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindService},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindNamespace},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindSecret},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindPolicy},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindIdentityProvider},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindAuthenticator},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindRegion},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindCredential},
		{api: ucorev1.API, version: ucorev1.Version, kind: ucorev1.KindGateway},
	}
}

func enterpriseResourceKinds() []resourceKind {
	return []resourceKind{
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindCollectorExporter},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindSecret},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindSecretStore},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindCertificate},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindCertificateIssuer},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindDNSProvider},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindDirectoryProvider},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindDirectoryProviderUser},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindDirectoryProviderGroup},
		{api: uenterprisev1.API, version: uenterprisev1.Version, kind: uenterprisev1.KindDeviceManager},
	}
}

func accessResourceKinds() []resourceKind {
	return []resourceKind{
		{api: uaccessv1.API, version: uaccessv1.Version, kind: uaccessv1.KindCatalog},
		{api: uaccessv1.API, version: uaccessv1.Version, kind: uaccessv1.KindPolicy},
		{api: uaccessv1.API, version: uaccessv1.Version, kind: uaccessv1.KindRequest},
		{api: uaccessv1.API, version: uaccessv1.Version, kind: uaccessv1.KindReview},
	}
}

func (s *Server) validateCoreReconcileKinds() error {
	client := reflect.ValueOf(s.octeliumC.CoreC())

	for _, rk := range coreResourceKinds() {
		methodName := fmt.Sprintf("List%s", rk.kind)
		if !client.MethodByName(methodName).IsValid() {
			return errors.Errorf("missing method %s", methodName)
		}

		if _, err := ucorev1.NewObject(rk.kind); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateEnterpriseReconcileKinds() error {
	client := reflect.ValueOf(s.octeliumC.EnterpriseC())

	for _, rk := range enterpriseResourceKinds() {
		methodName := fmt.Sprintf("List%s", rk.kind)
		if !client.MethodByName(methodName).IsValid() {
			return errors.Errorf("missing method %s", methodName)
		}

		if _, err := uenterprisev1.NewObject(rk.kind); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateAccessReconcileKinds() error {
	client := reflect.ValueOf(s.octeliumC.AccessC())

	for _, rk := range accessResourceKinds() {
		methodName := fmt.Sprintf("List%s", rk.kind)
		if !client.MethodByName(methodName).IsValid() {
			return errors.Errorf("missing method %s", methodName)
		}

		if _, err := uaccessv1.NewObject(rk.kind); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) startReconciliationLoop(ctx context.Context) {
	initial := time.NewTimer(30 * time.Second)
	defer initial.Stop()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-initial.C:
			if err := s.reconcileAllResources(ctx); err != nil {
				zap.L().Warn("Could not reconcile resources", zap.Error(err))
			}

		case <-ticker.C:
			if err := s.reconcileAllResources(ctx); err != nil {
				zap.L().Warn("Could not reconcile resources", zap.Error(err))
			}
		}
	}
}

func (s *Server) reconcileAllResources(ctx context.Context) error {
	for _, rk := range coreResourceKinds() {
		if err := s.reconcileCoreResourceKind(ctx, rk); err != nil {
			return errors.Errorf("could not reconcile %s/%s %s: %+v",
				rk.api, rk.version, rk.kind, err)
		}
	}

	for _, rk := range enterpriseResourceKinds() {
		if err := s.reconcileEnterpriseResourceKind(ctx, rk); err != nil {
			return errors.Errorf("could not reconcile %s/%s %s: %+v",
				rk.api, rk.version, rk.kind, err)
		}
	}

	for _, rk := range accessResourceKinds() {
		if err := s.reconcileAccessResourceKind(ctx, rk); err != nil {
			return errors.Errorf("could not reconcile %s/%s %s: %+v",
				rk.api, rk.version, rk.kind, err)
		}
	}

	go s.idxDebouncer.debounce()

	return nil
}

func (s *Server) reconcileCoreResourceKind(ctx context.Context, rk resourceKind) error {
	items, err := s.listAllCoreResourcesByKind(ctx, rk.kind)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(items))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		if item == nil || item.GetMetadata() == nil || item.GetMetadata().Uid == "" {
			continue
		}

		seen[item.GetMetadata().Uid] = struct{}{}

		if err := s.upsertResourceTx(ctx, tx, item); err != nil {
			return err
		}
	}

	localUIDs, err := s.listLocalUIDsTx(ctx, tx, rk.api, rk.version, rk.kind)
	if err != nil {
		return err
	}

	for _, uid := range localUIDs {
		if _, ok := seen[uid]; ok {
			continue
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM resources WHERE uid = ?`, uid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Server) listAllCoreResourcesByKind(
	ctx context.Context,
	kind string,
) ([]umetav1.ResourceObjectI, error) {
	client := reflect.ValueOf(s.octeliumC.CoreC())

	methodName := fmt.Sprintf("List%s", kind)
	method := client.MethodByName(methodName)
	if !method.IsValid() {
		return nil, errors.Errorf("unsupported core resource kind: %s", kind)
	}

	var ret []umetav1.ResourceObjectI
	var page uint32

	for {
		listOpts := &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: 500,
			Page:         page,
		}

		res := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(listOpts),
		})

		if len(res) != 2 {
			return nil, errors.Errorf("invalid %s return length: %d", methodName, len(res))
		}

		if !res[1].IsNil() {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.Errorf("%s returned non-error second value", methodName)
			}
			return nil, err
		}

		if res[0].IsNil() {
			return nil, errors.Errorf("%s returned nil response", methodName)
		}

		callRes, ok := res[0].Interface().(umetav1.ObjectI)
		if !ok {
			return nil, errors.Errorf("%s response does not implement umetav1.ObjectI", methodName)
		}

		retMap, err := pbutils.ConvertToMap(callRes)
		if err != nil {
			return nil, err
		}

		rawItems := retMap["items"]
		if rawItems != nil {
			retItems, ok := rawItems.([]any)
			if !ok {
				return nil, errors.Errorf("%s response items has invalid type", methodName)
			}

			for _, rawItem := range retItems {
				itemMap, ok := rawItem.(map[string]any)
				if !ok {
					return nil, errors.Errorf("%s response item has invalid type", methodName)
				}

				item, err := ucorev1.NewObject(kind)
				if err != nil {
					return nil, err
				}

				if err := pbutils.UnmarshalFromMap(itemMap, item); err != nil {
					return nil, err
				}

				ret = append(ret, item)
			}
		}

		hasMore := false
		if metaMap, ok := retMap["listResponseMeta"].(map[string]any); ok && metaMap != nil {
			hasMore, _ = metaMap["hasMore"].(bool)
		}

		if !hasMore {
			break
		}

		page++
	}

	return ret, nil
}

func (s *Server) reconcileEnterpriseResourceKind(ctx context.Context, rk resourceKind) error {
	items, err := s.listAllEnterpriseResourcesByKind(ctx, rk.kind)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(items))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		if item == nil || item.GetMetadata() == nil || item.GetMetadata().Uid == "" {
			continue
		}

		seen[item.GetMetadata().Uid] = struct{}{}

		if err := s.upsertResourceTx(ctx, tx, item); err != nil {
			return err
		}
	}

	localUIDs, err := s.listLocalUIDsTx(ctx, tx, rk.api, rk.version, rk.kind)
	if err != nil {
		return err
	}

	for _, uid := range localUIDs {
		if _, ok := seen[uid]; ok {
			continue
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM resources WHERE uid = ?`, uid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Server) listAllEnterpriseResourcesByKind(
	ctx context.Context,
	kind string,
) ([]umetav1.ResourceObjectI, error) {
	client := reflect.ValueOf(s.octeliumC.EnterpriseC())

	methodName := fmt.Sprintf("List%s", kind)
	method := client.MethodByName(methodName)
	if !method.IsValid() {
		return nil, errors.Errorf("unsupported enterprise resource kind: %s", kind)
	}

	var ret []umetav1.ResourceObjectI
	var page uint32

	for {
		listOpts := &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: 500,
			Page:         page,
		}

		res := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(listOpts),
		})

		if len(res) != 2 {
			return nil, errors.Errorf("invalid %s return length: %d", methodName, len(res))
		}

		if !res[1].IsNil() {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.Errorf("%s returned non-error second value", methodName)
			}
			return nil, err
		}

		if res[0].IsNil() {
			return nil, errors.Errorf("%s returned nil response", methodName)
		}

		callRes, ok := res[0].Interface().(umetav1.ObjectI)
		if !ok {
			return nil, errors.Errorf("%s response does not implement umetav1.ObjectI", methodName)
		}

		retMap, err := pbutils.ConvertToMap(callRes)
		if err != nil {
			return nil, err
		}

		rawItems := retMap["items"]
		if rawItems != nil {
			retItems, ok := rawItems.([]any)
			if !ok {
				return nil, errors.Errorf("%s response items has invalid type", methodName)
			}

			for _, rawItem := range retItems {
				itemMap, ok := rawItem.(map[string]any)
				if !ok {
					return nil, errors.Errorf("%s response item has invalid type", methodName)
				}

				item, err := uenterprisev1.NewObject(kind)
				if err != nil {
					return nil, err
				}

				if err := pbutils.UnmarshalFromMap(itemMap, item); err != nil {
					return nil, err
				}

				ret = append(ret, item)
			}
		}

		hasMore := false
		if metaMap, ok := retMap["listResponseMeta"].(map[string]any); ok && metaMap != nil {
			hasMore, _ = metaMap["hasMore"].(bool)
		}

		if !hasMore {
			break
		}

		page++
	}

	return ret, nil
}

func (s *Server) reconcileAccessResourceKind(ctx context.Context, rk resourceKind) error {
	items, err := s.listAllAccessResourcesByKind(ctx, rk.kind)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(items))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		if item == nil || item.GetMetadata() == nil || item.GetMetadata().Uid == "" {
			continue
		}

		seen[item.GetMetadata().Uid] = struct{}{}

		if err := s.upsertResourceTx(ctx, tx, item); err != nil {
			return err
		}
	}

	localUIDs, err := s.listLocalUIDsTx(ctx, tx, rk.api, rk.version, rk.kind)
	if err != nil {
		return err
	}

	for _, uid := range localUIDs {
		if _, ok := seen[uid]; ok {
			continue
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM resources WHERE uid = ?`, uid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Server) listAllAccessResourcesByKind(
	ctx context.Context,
	kind string,
) ([]umetav1.ResourceObjectI, error) {
	client := reflect.ValueOf(s.octeliumC.AccessC())

	methodName := fmt.Sprintf("List%s", kind)
	method := client.MethodByName(methodName)
	if !method.IsValid() {
		return nil, errors.Errorf("unsupported access resource kind: %s", kind)
	}

	var ret []umetav1.ResourceObjectI
	var page uint32

	for {
		listOpts := &rmetav1.ListOptions{
			Paginate:     true,
			ItemsPerPage: 500,
			Page:         page,
		}

		res := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(listOpts),
		})

		if len(res) != 2 {
			return nil, errors.Errorf("invalid %s return length: %d", methodName, len(res))
		}

		if !res[1].IsNil() {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.Errorf("%s returned non-error second value", methodName)
			}
			return nil, err
		}

		if res[0].IsNil() {
			return nil, errors.Errorf("%s returned nil response", methodName)
		}

		callRes, ok := res[0].Interface().(umetav1.ObjectI)
		if !ok {
			return nil, errors.Errorf("%s response does not implement umetav1.ObjectI", methodName)
		}

		retMap, err := pbutils.ConvertToMap(callRes)
		if err != nil {
			return nil, err
		}

		rawItems := retMap["items"]
		if rawItems != nil {
			retItems, ok := rawItems.([]any)
			if !ok {
				return nil, errors.Errorf("%s response items has invalid type", methodName)
			}

			for _, rawItem := range retItems {
				itemMap, ok := rawItem.(map[string]any)
				if !ok {
					return nil, errors.Errorf("%s response item has invalid type", methodName)
				}

				item, err := uaccessv1.NewObject(kind)
				if err != nil {
					return nil, err
				}

				if err := pbutils.UnmarshalFromMap(itemMap, item); err != nil {
					return nil, err
				}

				ret = append(ret, item)
			}
		}

		hasMore := false
		if metaMap, ok := retMap["listResponseMeta"].(map[string]any); ok && metaMap != nil {
			hasMore, _ = metaMap["hasMore"].(bool)
		}

		if !hasMore {
			break
		}

		page++
	}

	return ret, nil
}

func (s *Server) listLocalUIDsTx(
	ctx context.Context,
	tx *sql.Tx,
	api string,
	version string,
	kind string,
) ([]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT uid FROM resources WHERE api = ? AND version = ? AND kind = ?`,
		api,
		version,
		kind,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ret []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		ret = append(ret, uid)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *Server) upsertResource(ctx context.Context, rsc umetav1.ResourceObjectI) error {
	return s.upsertResourceExec(ctx, s.db, rsc)
}

func (s *Server) upsertResourceTx(ctx context.Context, tx *sql.Tx, rsc umetav1.ResourceObjectI) error {
	return s.upsertResourceExec(ctx, tx, rsc)
}

func (s *Server) upsertResourceExec(ctx context.Context, e execer, rsc umetav1.ResourceObjectI) error {
	if rsc == nil || rsc.GetMetadata() == nil {
		return nil
	}

	rscJSON, err := pbutils.MarshalJSON(rsc, false)
	if err != nil {
		return err
	}

	api, version := vutils.SplitApiVersion(rsc.GetApiVersion())
	kind := rsc.GetKind()
	uid := rsc.GetMetadata().Uid
	resourceVersion := rsc.GetMetadata().ResourceVersion

	rscStr, err := s.getRSCStr(rscJSON)
	if err != nil {
		return err
	}

	_, err = e.ExecContext(ctx, `
INSERT INTO resources
    (api, version, kind, uid, resource_version, rsc, rsc_str)
VALUES
    (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (uid)
DO UPDATE SET
    api = EXCLUDED.api,
    version = EXCLUDED.version,
    kind = EXCLUDED.kind,
    resource_version = EXCLUDED.resource_version,
    rsc = EXCLUDED.rsc,
    rsc_str = EXCLUDED.rsc_str
`,
		api,
		version,
		kind,
		uid,
		resourceVersion,
		string(rscJSON),
		rscStr,
	)

	return err
}
