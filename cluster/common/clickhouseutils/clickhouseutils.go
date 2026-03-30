// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package clickhouseutils

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/octelium/octelium/pkg/utils/ldflags"
	"go.uber.org/zap"
)

type ClickHouseConnOpt struct {
}

func GetConn(ctx context.Context, o *ClickHouseConnOpt) (driver.Conn, error) {
	var (
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{func() string {
				if ldflags.IsTest() {
					return "localhost:9000"
				}
				return "octelium-clickhouse.octelium.svc:9000"
			}()},
			Auth: clickhouse.Auth{
				Database: "",
				Username: "octelium",
				Password: "octelium",
			},

			Debugf: zap.S().Debugf,
		})
	)

	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	{
		err := conn.Exec(ctx, "SET enable_json_type = 1")
		if err != nil {
			return nil, err
		}

		err = conn.Exec(ctx, "SET allow_experimental_json_type = 1")
		if err != nil {
			return nil, err
		}

		err = conn.Exec(ctx, "SET output_format_native_write_json_as_string = 1")
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}
