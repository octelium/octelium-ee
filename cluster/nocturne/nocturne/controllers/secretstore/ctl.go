// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package usrcontroller

import (
	"context"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
	"github.com/octelium/octelium/apis/main/enterprisev1"
)

type Controller struct {
	octeliumC octeliumc.ClientInterface
}

func NewController(
	octeliumC octeliumc.ClientInterface,
) *Controller {
	return &Controller{
		octeliumC: octeliumC,
	}
}

func (c *Controller) OnAdd(ctx context.Context, crt *enterprisev1.SecretStore) error {

	return nil
}

func (c *Controller) OnUpdate(ctx context.Context, new, old *enterprisev1.SecretStore) error {

	return nil
}

func (c *Controller) OnDelete(ctx context.Context, usr *enterprisev1.SecretStore) error {

	return nil
}
