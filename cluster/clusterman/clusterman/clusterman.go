// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package clusterman

/*
import (
	"context"
	"os"
	"os/signal"

	"go.uber.org/zap"

	"github.com/octelium/octelium-ee/cluster/common/octeliumc"
)

func Run() error {
	ctx, cancelFn := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFn()

	_, err := octeliumc.NewClient(ctx, nil)
	if err != nil {
		return err
	}

	zap.S().Debugf("Cluster Manager is running...")

	<-ctx.Done()

	return nil
}
*/
