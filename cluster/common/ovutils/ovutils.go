// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package ovutils

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uenterprisev1"
	"github.com/octelium/octelium/apis/cluster/cbootstrapv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/octelium/octelium/pkg/apiutils/ucorev1"
	"github.com/octelium/octelium/pkg/apiutils/umetav1"
	"github.com/octelium/octelium/pkg/common/pbutils"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

func NewResourceObject(api, version, kind string) (umetav1.ResourceObjectI, error) {
	switch api {
	case ucorev1.API:
		return ucorev1.NewObject(kind)
	case uenterprisev1.API:
		return uenterprisev1.NewObject(kind)
	default:
		return nil, errors.Errorf("Invalid API: %s", api)
	}
}

func NewResourceObjectList(api, version, kind string) (proto.Message, error) {
	switch api {
	case ucorev1.API:
		return ucorev1.NewObjectList(kind)
	case uenterprisev1.API:
		return uenterprisev1.NewObjectList(kind)
	default:
		return nil, errors.Errorf("Invalid API: %s", api)
	}
}

func GetBootstrapConfig(ctx context.Context, octeliumC octeliumc.ClientInterface) (*cbootstrapv1.Config, error) {

	sec, err := octeliumC.CoreC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: "sys:bootstrap-config",
	})
	if err != nil {
		return nil, err
	}

	ret := &cbootstrapv1.Config{}

	if err := pbutils.Unmarshal(ucorev1.ToSecret(sec).GetValueBytes(), ret); err != nil {
		return nil, err
	}

	return ret, nil
}

const ExtInfoKeyEnterprise = "enterpriseV1"
const DirectoryProviderSessionPrefix = "sys:dp"

func GetOIDCConfigSecretName(regionName string) string {
	return fmt.Sprintf("sys:oidc-config-%s", regionName)
}

func GetOIDC_JWKSSecretName(regionName string) string {
	return fmt.Sprintf("sys:oidc-jwks-%s", regionName)
}

var isMock bool

func SetMockMode() {
	isMock = true
}

func IsMockMode() bool {
	return isMock
}

func IsPrivateRegistry() bool {
	return true
}

func getDuckDBDir() string {
	return "/octelium-data"
}

func GetDuckDBDSN() string {
	if ldflags.IsTest() {

		dir, _ := os.MkdirTemp("", "duckdb-*")
		return fmt.Sprintf(`%s?extension_directory=%s`, path.Join(dir, "store.db"), dir)
	}

	dir := func() string {
		if val := os.Getenv("OCTELIUM_DUCKDB_PATH"); val != "" {
			return val
		}

		return getDuckDBDir()
	}()

	return fmt.Sprintf(`%s?extension_directory=%s`, path.Join(dir, "store.db"), dir)
}
