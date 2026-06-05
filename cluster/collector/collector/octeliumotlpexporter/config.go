// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package octeliumotlpexporter

import (
	"errors"
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type Config struct {
	TimeoutConfig exporterhelper.TimeoutConfig                             `mapstructure:",squash"`
	QueueConfig   configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`

	Endpoint              string            `mapstructure:"endpoint"`
	Headers               map[string]string `mapstructure:"headers"`
	Compression           string            `mapstructure:"compression"`
	WaitForReady          bool              `mapstructure:"wait_for_ready"`
	MaxCallRecvMsgSizeMiB int               `mapstructure:"max_call_recv_msg_size_mib"`
	MaxCallSendMsgSizeMiB int               `mapstructure:"max_call_send_msg_size_mib"`

	_ struct{}
}

var (
	_ component.Config  = (*Config)(nil)
	_ confmap.Validator = (*Config)(nil)
)

func (c *Config) Validate() error {
	if c.sanitizedEndpoint() == "" {
		return errors.New(`requires a non-empty "endpoint"`)
	}

	if c.MaxCallRecvMsgSizeMiB < 0 {
		return errors.New("max_call_recv_msg_size_mib cannot be negative")
	}

	if c.MaxCallSendMsgSizeMiB < 0 {
		return errors.New("max_call_send_msg_size_mib cannot be negative")
	}

	switch c.Compression {
	case "", "gzip":
	default:
		return errors.New(`compression must be empty or "gzip"`)
	}

	return nil
}

func (c *Config) sanitizedEndpoint() string {
	switch {
	case strings.HasPrefix(c.Endpoint, "http://"):
		return strings.TrimPrefix(c.Endpoint, "http://")
	case strings.HasPrefix(c.Endpoint, "https://"):
		return strings.TrimPrefix(c.Endpoint, "https://")
	case strings.HasPrefix(c.Endpoint, "dns://"):
		r := regexp.MustCompile(`^dns:///?`)
		return r.ReplaceAllString(c.Endpoint, "")
	default:
		return c.Endpoint
	}
}
