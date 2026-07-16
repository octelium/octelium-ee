module github.com/octelium/octelium-ee/cluster/collector

go 1.26.4

require (
	github.com/octelium/octelium-ee/cluster/common v0.0.0-00010101000000-000000000000
	github.com/octelium/octelium-ee/pkg v0.0.0-00010101000000-000000000000
	github.com/octelium/octelium/apis v0.0.0-00010101000000-000000000000
	github.com/octelium/octelium/cluster/common v0.0.0-20260716025311-aaaa13c8927a
	github.com/octelium/octelium/pkg v0.0.0-20260716025311-aaaa13c8927a
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuredataexplorerexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.153.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/intervalprocessor v0.153.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.59.0
	go.opentelemetry.io/collector/config/configcompression v1.59.0
	go.opentelemetry.io/collector/config/configgrpc v0.153.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.59.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.59.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.59.0
	go.opentelemetry.io/collector/confmap v1.59.0
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.59.0
	go.opentelemetry.io/collector/consumer v1.59.0
	go.opentelemetry.io/collector/consumer/consumererror v0.153.0
	go.opentelemetry.io/collector/exporter v1.59.0
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.153.0 // indirect
	go.opentelemetry.io/collector/exporter/otlpexporter v0.153.0
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.153.0
	go.opentelemetry.io/collector/exporter/xexporter v0.153.0 // indirect
	go.opentelemetry.io/collector/extension v1.59.0 // indirect
	go.opentelemetry.io/collector/otelcol v0.153.0
	go.opentelemetry.io/collector/pdata v1.59.0
	go.opentelemetry.io/collector/pdata/pprofile v0.153.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.59.0 // indirect
	go.opentelemetry.io/collector/processor v1.59.0
	go.opentelemetry.io/collector/processor/batchprocessor v0.153.0
	go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.153.0
	go.opentelemetry.io/collector/receiver v1.59.0
	go.opentelemetry.io/collector/service v0.153.0
	go.uber.org/zap v1.28.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478
	google.golang.org/grpc v1.82.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension v0.153.0
	go.opentelemetry.io/collector/component/componentstatus v0.153.0
	go.opentelemetry.io/collector/config/configoptional v1.59.0
	go.opentelemetry.io/collector/exporter/exporterhelper v0.153.0
	go.opentelemetry.io/collector/receiver/receiverhelper v0.153.0
	go.opentelemetry.io/otel v1.43.0
)

require (
	cloud.google.com/go/auth v0.18.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	code.cloudfoundry.org/clock v1.0.0 // indirect
	github.com/Azure/azure-kusto-go/azkustodata v1.2.2 // indirect
	github.com/Azure/azure-kusto-go/azkustoingest v1.2.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/data/aztables v1.4.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.3 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue v1.0.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.1 // indirect
	github.com/ClickHouse/ch-go v0.71.0 // indirect
	github.com/ClickHouse/clickhouse-go/v2 v2.46.0 // indirect
	github.com/GehirnInc/crypt v0.0.0-20230320061759-8cc1b52080c5 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Showmax/go-fqdn v1.0.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.4 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/antchfx/xmlquery v1.5.1 // indirect
	github.com/antchfx/xpath v1.3.6 // indirect
	github.com/apache/thrift v0.23.1-0.20260429145742-d2acd3c49e58 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-msk-iam-sasl-signer-go v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.7 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.17 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager v0.1.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.101.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.21.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/doug-martin/goqu/v9 v9.19.0 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.8.0 // indirect
	github.com/elastic/go-docappender/v2 v2.14.1 // indirect
	github.com/elastic/go-freelru v0.16.0 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/elastic/go-structform v0.0.12 // indirect
	github.com/elastic/lunes v0.2.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/expr-lang/expr v1.17.8 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20251226215517-609e4778396f // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.14 // indirect
	github.com/googleapis/gax-go/v2 v2.18.0 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/influxdb-observability/common v0.5.12 // indirect
	github.com/influxdata/influxdb-observability/otel2influx v0.5.12 // indirect
	github.com/influxdata/line-protocol/v2 v2.2.1 // indirect
	github.com/itchyny/timefmt-go v0.1.8 // indirect
	github.com/jaegertracing/jaeger-idl v0.7.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.7.7 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.4 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lestrrat-go/strftime v1.1.1 // indirect
	github.com/lib/pq v1.11.2 // indirect
	github.com/lufia/plan9stats v0.0.0-20251013123823-9fd1530e3ec3 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/microsoft/ApplicationInsights-Go v0.4.4 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/octelium/octelium/cluster/rscserver v0.0.0-20260716025311-aaaa13c8927a // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/internal/basicauth v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/internal/credentialsfile v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/sumologicextension v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/exp/metrics v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchperresourceattr v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/core/xidutils v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/topic v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheusremotewrite v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/splunk v0.153.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.153.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/paulmach/orb v0.12.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_golang/exp v0.0.0-20260325093428-d8591d0db856 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/prometheus/prometheus v1.8.2-0.20220303173753-edfe657b5405 // indirect
	github.com/prometheus/sigv4 v0.4.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/samber/lo v1.52.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shirou/gopsutil/v4 v4.26.4 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tg123/go-htpasswd v1.2.4 // indirect
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/tinylru v1.1.0 // indirect
	github.com/tidwall/wal v1.2.1 // indirect
	github.com/tilinna/clock v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/twmb/franz-go v1.21.2 // indirect
	github.com/twmb/franz-go/pkg/kadm v1.18.0 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.13.1 // indirect
	github.com/twmb/franz-go/pkg/sasl/kerberos v1.1.0 // indirect
	github.com/twmb/franz-go/plugin/kzap v1.1.2 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/ua-parser/uap-go v0.0.0-20251207011819-db9adb27a0b8 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.elastic.co/fastjson v1.5.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector v0.153.0 // indirect
	go.opentelemetry.io/collector/client v1.59.0 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.153.0 // indirect
	go.opentelemetry.io/collector/config/configauth v1.59.0 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.153.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v1.59.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.153.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.59.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.153.0 // indirect
	go.opentelemetry.io/collector/connector v0.153.0 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.153.0 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.153.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.153.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.153.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.153.0 // indirect
	go.opentelemetry.io/collector/exporter/exportertest v0.153.0 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.59.0 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.153.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.153.0 // indirect
	go.opentelemetry.io/collector/extension/extensiontest v0.153.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.153.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.59.0 // indirect
	go.opentelemetry.io/collector/internal/componentalias v0.153.0 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.153.0 // indirect
	go.opentelemetry.io/collector/internal/memorylimiter v0.153.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.153.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.153.0 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.153.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.153.0 // indirect
	go.opentelemetry.io/collector/processor/processorhelper v0.153.0 // indirect
	go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper v0.153.0 // indirect
	go.opentelemetry.io/collector/processor/processortest v0.153.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.153.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.153.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.153.0 // indirect
	go.opentelemetry.io/collector/semconv v0.128.1-0.20250610090210-188191247685 // indirect
	go.opentelemetry.io/collector/service/hostcapabilities v0.153.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.18.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.23.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.43.0 // indirect
	go.opentelemetry.io/ebpf-profiler v0.0.202618 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.65.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0 // indirect
	go.opentelemetry.io/otel/log v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.19.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	google.golang.org/api v0.272.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.35.3 // indirect
	k8s.io/apimachinery v0.35.3 // indirect
	k8s.io/client-go v0.35.3 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	k8s.io/utils v0.0.0-20260319190234-28399d86e0b5 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

// replace cloud.google.com/go => cloud.google.com/go v0.100.2

// replace github.com/Azure/azure-sdk-for-go/sdk/storage/azblob => github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v0.3.0

// replace github.com/Azure/azure-kusto-go => github.com/Azure/azure-kusto-go v0.10.2

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v0.305.1-0.20250808193045-294f36e80261

replace github.com/octelium/octelium/apis => ../../apis

replace github.com/octelium/octelium-ee/cluster/common => ../common

replace github.com/octelium/octelium-ee/pkg => ../../pkg

replace github.com/octelium/octelium-ee/cluster/rscserver => ../rscserver

// replace cloud.google.com/go => cloud.google.com/go v0.110.9

// replace go.opentelemetry.io/otel/log => go.opentelemetry.io/otel/log v0.7.0

replace github.com/knadh/koanf => github.com/knadh/koanf v1.5.0
