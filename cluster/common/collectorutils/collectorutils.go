// Copyright (c) 2025-present Octelium Labs, LLC. All rights reserved.
//
// This software is licensed under the Octelium Enterprise Source-Available License.
// Commercial and production use is strictly prohibited without a valid
// Commercial Agreement from Octelium Labs, LLC.
//
// See the LICENSE file in the repository root for full license text.

package collectorutils

import (
	"fmt"
	"strings"
)

const ServiceName = "collector.octelium"

const IngestPort = 8081

const (
	LogsServiceFullName    = "opentelemetry.proto.collector.logs.v1.LogsService"
	MetricsServiceFullName = "opentelemetry.proto.collector.metrics.v1.MetricsService"
	ExportMethod           = "Export"
)

func GetServiceEndpoint(domain string) string {
	return fmt.Sprintf("%s.%s:443", ServiceName, strings.TrimSuffix(domain, "."))
}

func GetIngestAddr() string {
	return fmt.Sprintf(":%d", IngestPort)
}

func GetUpstreamURL(host string) string {
	return fmt.Sprintf("grpc://%s:%d", host, IngestPort)
}
