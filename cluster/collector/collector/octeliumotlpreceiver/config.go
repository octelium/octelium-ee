// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpreceiver

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	Endpoint             string `mapstructure:"endpoint"`
	MaxRecvMsgSizeMiB    int    `mapstructure:"max_recv_msg_size_mib"`
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams"`
}

var _ component.Config = (*Config)(nil)

func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	if c.MaxRecvMsgSizeMiB < 0 {
		return errors.New("max_recv_msg_size_mib cannot be negative")
	}
	return nil
}