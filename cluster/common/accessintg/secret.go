// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package accessintg

import (
	"context"
	"strings"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium-ee/pkg/apiutils/uaccessv1"
	"github.com/octelium/octelium/apis/rsc/rmetav1"
	"github.com/pkg/errors"
)

func GetSecretValue(ctx context.Context, octeliumC octeliumc.ClientInterface,
	name string) (string, error) {
	if name == "" {
		return "", errors.Errorf("Empty Secret name")
	}

	sec, err := octeliumC.AccessC().GetSecret(ctx, &rmetav1.GetOptions{
		Name: name,
	})
	if err != nil {
		return "", err
	}

	val := strings.TrimSpace(uaccessv1.ToSecret(sec).GetValueStr())
	if val == "" {
		return "", errors.Errorf("The Secret %s is empty", name)
	}

	return val, nil
}
