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
	Endpoint string `mapstructure:"endpoint"`

	MaxRecvMsgSizeMiB    int    `mapstructure:"max_recv_msg_size_mib"`
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams"`

	ReadBufferSize  int `mapstructure:"read_buffer_size"`
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	_ struct{}
}

var _ component.Config = (*Config)(nil)

func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	if c.MaxRecvMsgSizeMiB < 0 {
		return errors.New("max_recv_msg_size_mib cannot be negative")
	}

	if c.MaxConcurrentStreams == 0 {
		return errors.New("max_concurrent_streams cannot be zero")
	}

	if c.ReadBufferSize < 0 {
		return errors.New("read_buffer_size cannot be negative")
	}

	if c.WriteBufferSize < 0 {
		return errors.New("write_buffer_size cannot be negative")
	}

	return nil
}
