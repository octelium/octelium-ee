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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	otests "github.com/octelium/octelium-ee/cluster/common/tests"
	"github.com/octelium/octelium/apis/main/corev1"
	"github.com/octelium/octelium/apis/main/metav1"
	"github.com/octelium/octelium/cluster/common/vutils"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/utilrand"
	"github.com/stretchr/testify/assert"
)

type rscStoreTestEnv struct {
	ctx context.Context
	srv *Server
}

func newRscStoreTestEnv(t *testing.T) *rscStoreTestEnv {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		return nil
	}

	srv, err := newServer(ctx, tst.C.OcteliumC)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		tst.Destroy()
		return nil
	}

	err = srv.initDB(ctx)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		cancel()
		_ = srv.db.Close()
		tst.Destroy()
		return nil
	}

	srv.idxDebouncer.after = time.Hour

	t.Cleanup(func() {
		cancel()

		srv.idxDebouncer.mu.Lock()
		if srv.idxDebouncer.timer != nil {
			srv.idxDebouncer.timer.Stop()
		}
		srv.idxDebouncer.mu.Unlock()

		assert.Nil(t, srv.db.Close())
		tst.Destroy()
	})

	return &rscStoreTestEnv{
		ctx: ctx,
		srv: srv,
	}
}

func newRscStoreMetadata(name string, createdAt time.Time) *metav1.Metadata {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &metav1.Metadata{
		Name:            name,
		Uid:             vutils.UUIDv4(),
		ResourceVersion: vutils.UUIDv7(),
		CreatedAt:       pbutils.Timestamp(createdAt),
	}
}

func insertRscStoreObject(t *testing.T, env *rscStoreTestEnv, rsc umetav1.ResourceObjectI) {
	t.Helper()

	err := env.srv.upsertResource(env.ctx, rsc)
	assert.Nil(t, err, "%+v", err)
}

func insertRawRscStoreResource(t *testing.T, env *rscStoreTestEnv, kind string, rsc map[string]any) {
	t.Helper()

	metadata, ok := rsc["metadata"].(map[string]any)
	assert.True(t, ok)
	if !ok {
		return
	}

	uid, _ := metadata["uid"].(string)
	resourceVersion, _ := metadata["resourceVersion"].(string)

	rscJSON, err := json.Marshal(rsc)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	rscStr, err := env.srv.getRSCStr(rscJSON)
	assert.Nil(t, err, "%+v", err)
	if err != nil {
		return
	}

	_, err = env.srv.db.ExecContext(env.ctx, `
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
		ucorev1.API,
		ucorev1.Version,
		kind,
		uid,
		resourceVersion,
		string(rscJSON),
		rscStr,
	)
	assert.Nil(t, err, "%+v", err)
}

func newRawRscStoreResource(kind, name string, hidden bool, spec, status map[string]any) map[string]any {
	if spec == nil {
		spec = map[string]any{}
	}
	if status == nil {
		status = map[string]any{}
	}

	return map[string]any{
		"apiVersion": ucorev1.APIVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"name":            name,
			"uid":             vutils.UUIDv4(),
			"resourceVersion": vutils.UUIDv7(),
			"createdAt":       time.Now().UTC().Format(time.RFC3339Nano),
			"isSystemHidden":  hidden,
		},
		"spec":   spec,
		"status": status,
	}
}

func getStoredRscStoreResource(t *testing.T, env *rscStoreTestEnv, uid string) map[string]any {
	t.Helper()

	ret := make(map[string]any)
	err := env.srv.db.QueryRowContext(env.ctx, `SELECT rsc FROM resources WHERE uid = ?`, uid).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func getStoredRscStoreResourceVersion(t *testing.T, env *rscStoreTestEnv, uid string) string {
	t.Helper()

	var ret string
	err := env.srv.db.QueryRowContext(env.ctx, `SELECT resource_version FROM resources WHERE uid = ?`, uid).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func getRscStoreResourceCount(t *testing.T, env *rscStoreTestEnv, kind string) int {
	t.Helper()

	var ret int
	err := env.srv.db.QueryRowContext(env.ctx, `SELECT COUNT(*) FROM resources WHERE api = ? AND version = ? AND kind = ?`,
		ucorev1.API, ucorev1.Version, kind).Scan(&ret)
	assert.Nil(t, err, "%+v", err)

	return ret
}

func TestTransformRsc(t *testing.T) {
	ctx := context.Background()
	tst, err := otests.Initialize(nil)
	assert.Nil(t, err, "%+v", err)
	t.Cleanup(func() {
		tst.Destroy()
	})
	fakeC := tst.C

	srv, err := newServer(ctx, fakeC.OcteliumC)
	assert.Nil(t, err)

	{
		rsc := &corev1.User{
			ApiVersion: ucorev1.APIVersion,
			Kind:       ucorev1.KindUser,
			Metadata: &metav1.Metadata{
				Name:            utilrand.GetRandomStringCanonical(8),
				Uid:             vutils.UUIDv4(),
				ResourceVersion: vutils.UUIDv7(),
				CreatedAt:       pbutils.Timestamp(time.Now().UTC().Add(-time.Duration(utilrand.GetRandomRangeMath(1, 500) * int(time.Minute)))),
			},
			Spec: &corev1.User_Spec{
				Type: func() corev1.User_Spec_Type {
					if utilrand.GetRandomRangeMath(1, 500)%2 == 0 {
						return corev1.User_Spec_HUMAN
					}
					return corev1.User_Spec_WORKLOAD
				}(),
				Email: fmt.Sprintf("%s@example.com", utilrand.GetRandomStringCanonical(9)),
			},
			Status: &corev1.User_Status{},
		}

		rscJSON, err := pbutils.MarshalJSON(rsc, false)
		assert.Nil(t, err)
		_, err = srv.getRSCStr(rscJSON)
		assert.Nil(t, err)

	}

}
