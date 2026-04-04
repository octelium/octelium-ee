module github.com/octelium/octelium-ee/cluster/rscserver

go 1.25.8

require (
	github.com/octelium/octelium-ee/cluster/common v0.0.0-00010101000000-000000000000
	github.com/octelium/octelium-ee/pkg v0.0.0-00010101000000-000000000000
	github.com/octelium/octelium/apis v0.0.0-00010101000000-000000000000
	github.com/octelium/octelium/cluster/common v0.0.0-20260403181525-3d68fc772d1a
	github.com/octelium/octelium/cluster/rscserver v0.0.0-20260403181525-3d68fc772d1a
	github.com/octelium/octelium/pkg v0.0.0-20260403181525-3d68fc772d1a
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.8.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.79.3
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/doug-martin/goqu/v9 v9.19.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lib/pq v1.11.2 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.40.0 // indirect
	go.opentelemetry.io/otel/log v0.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2 // indirect
)

replace github.com/octelium/octelium/apis => ../../apis

replace github.com/octelium/octelium-ee/cluster/common => ../common

replace github.com/octelium/octelium-ee/pkg => ../../pkg

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20241209162323-e6fa225c2576
