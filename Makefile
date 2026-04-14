
export GO111MODULE=on
export PATH := $(PATH):$(shell go env GOPATH)/bin

.PHONY: gen-api build-cli build-octelium clean fmt lint test unit vendor

REPOSITORY := github.com/octelium/octelium
REGISTRY ?= ghcr.io
IMAGE_PREFIX := octelium

COMMIT := $(shell git rev-parse HEAD)
TAG := $(shell git describe --tags --exact-match $(COMMIT) 2>/dev/null)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS_PATH := $(REPOSITORY)/pkg/utils/ldflags

LDF_IMAGE_REGISTRY := $(LDFLAGS_PATH).ImageRegistry=$(REGISTRY)
LDF_IMAGE_REGISTRY_PREFIX := $(LDFLAGS_PATH).ImageRegistryPrefix=$(IMAGE_PREFIX)
LDF_COMMIT := $(LDFLAGS_PATH).GitCommit=$(COMMIT)
LDF_TAG := $(LDFLAGS_PATH).GitTag=$(TAG)
LDF_SEMVER := $(LDFLAGS_PATH).SemVer=$(TAG)
LDF_BRANCH := $(LDFLAGS_PATH).GitBranch=$(BRANCH)

GENERATED_API_DOCS_DIR := ./tmp/docs/apis
GENERATED_API_DOCS_TEMP := ./unsorted/protoc/template.tmpl
GO_BIN_DIR := $${HOME}/go/bin

LDFLAGS := -ldflags '-X $(LDF_COMMIT) -X $(LDF_TAG) -X $(LDF_BRANCH) -X $(LDF_SEMVER)\
-X $(LDF_IMAGE_REGISTRY) -X $(LDF_IMAGE_REGISTRY_PREFIX)'

PROTO_GO_OPT := --go_opt=paths=source_relative
PROTO_GO_OPT_GRPC := $(PROTO_GO_OPT) --go-grpc_opt=paths=source_relative
PROTO_IN_PREFIX := apis/protobuf
PROTO_IN_MAIN := $(PROTO_IN_PREFIX)/main
PROTO_IN_CLUSTER := $(PROTO_IN_PREFIX)/cluster
PROTO_IN_CLIENT := $(PROTO_IN_PREFIX)/client
PROTO_IN_RSC := $(PROTO_IN_PREFIX)/rsc

CMD_TIDY := go mod tidy

build-rscserver:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-rscserver github.com/octelium/octelium-ee/cluster/rscserver
build-apiserver:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-apiserver github.com/octelium/octelium-ee/cluster/apiserver
build-nocturne:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-nocturne github.com/octelium/octelium-ee/cluster/nocturne
build-cloudman:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-cloudman github.com/octelium/octelium-ee/cluster/cloudman
build-collector:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-collector github.com/octelium/octelium-ee/cluster/collector
build-secretman:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-secretman github.com/octelium/octelium-ee/cluster/secretman
build-dirsync:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-dirsync github.com/octelium/octelium-ee/cluster/dirsync
build-genesis:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-genesis github.com/octelium/octelium-ee/cluster/genesis
build-clusterman:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-clusterman github.com/octelium/octelium-ee/cluster/clusterman
build-console:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-console github.com/octelium/octelium-ee/cluster/console
build-mockapiserver:
	CGO_ENABLED=1 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-mockapiserver github.com/octelium/octelium-ee/cluster/mockapiserver
build-publicserver:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-publicserver github.com/octelium/octelium-ee/cluster/publicserver
build-logstore:
	CGO_ENABLED=1 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-logstore github.com/octelium/octelium-ee/cluster/logstore
build-rscstore:
	CGO_ENABLED=1 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-rscstore github.com/octelium/octelium-ee/cluster/rscstore
build-metricstore:
	CGO_ENABLED=1 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-metricstore github.com/octelium/octelium-ee/cluster/metricstore
build-policyportal:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-policyportal github.com/octelium/octelium-ee/cluster/policyportal
build-e2e:
	CGO_ENABLED=0 GOOS=linux go build $(LDFLAGS) -o bin/octeliumee-e2e github.com/octelium/octelium-ee/cluster/e2e

protoc-install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.1
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
	mkdir -p $(GENERATED_API_DOCS_DIR)


gen-go-main:
	mkdir -p apis/main/metav1 apis/main/corev1 apis/main/authv1 apis/main/userv1
	mkdir -p apis/main/enterprisev1
	mkdir -p apis/main/accessv1 apis/main/visibilityv1
	mkdir -p apis/main/visibilityv1/vmetav1
	mkdir -p apis/main/visibilityv1/vcorev1
	protoc -I . -I $(PROTO_IN_MAIN)/metav1 metav1.proto \
		--go_out=apis/main/metav1 --go-grpc_out=apis/main/metav1 $(PROTO_GO_OPT)
	protoc -I . -I $(PROTO_IN_MAIN)/corev1 corev1.proto \
		--go_out=apis/main/corev1 --go-grpc_out=apis/main/corev1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_MAIN)/userv1 userv1.proto \
		--go_out=apis/main/userv1 --go-grpc_out=apis/main/userv1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_MAIN)/authv1 authv1.proto \
		--go_out=apis/main/authv1 --go-grpc_out=apis/main/authv1 $(PROTO_GO_OPT_GRPC)
	
	protoc -I . -I $(PROTO_IN_MAIN)/enterprisev1 enterprisev1.proto \
		--go_out=apis/main/enterprisev1 --go-grpc_out=apis/main/enterprisev1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_MAIN)/accessv1 accessv1.proto \
		--go_out=apis/main/accessv1 --go-grpc_out=apis/main/accessv1 $(PROTO_GO_OPT_GRPC)

	

	protoc -I . -I $(PROTO_IN_MAIN)/visibilityv1 visibilityv1.proto \
		--go_out=apis/main/visibilityv1 --go-grpc_out=apis/main/visibilityv1 $(PROTO_GO_OPT_GRPC)
	
	protoc -I . -I $(PROTO_IN_MAIN)/visibilityv1/meta vmetav1.proto \
		--go_out=apis/main/visibilityv1/vmetav1 --go-grpc_out=apis/main/visibilityv1/vmetav1 $(PROTO_GO_OPT_GRPC)

	protoc -I . -I $(PROTO_IN_MAIN)/visibilityv1/core vcorev1.proto \
		--go_out=apis/main/visibilityv1/vcorev1 --go-grpc_out=apis/main/visibilityv1/vcorev1 $(PROTO_GO_OPT_GRPC)

gen-go-cluster:
	mkdir -p apis/cluster/cclusterv1 apis/cluster/coctovigilv1
	mkdir -p apis/cluster/csecretmanv1
	protoc -I . -I $(PROTO_IN_CLUSTER)/clusterv1 cclusterv1.proto \
		--go_out=apis/cluster/cclusterv1 --go-grpc_out=apis/cluster/cclusterv1 $(PROTO_GO_OPT)
	protoc -I . -I $(PROTO_IN_CLUSTER)/octovigilv1 coctovigilv1.proto \
		--go_out=apis/cluster/coctovigilv1 --go-grpc_out=apis/cluster/coctovigilv1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_CLUSTER)/secretmanv1 csecretmanv1.proto \
		--go_out=apis/cluster/csecretmanv1 --go-grpc_out=apis/cluster/csecretmanv1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_CLUSTER)/bootstrapv1 cbootstrapv1.proto \
		--go_out=apis/cluster/cbootstrapv1 --go-grpc_out=apis/cluster/cbootstrapv1 $(PROTO_GO_OPT)

gen-go-rsc:
	mkdir -p apis/rsc/rmetav1 apis/rsc/rcorev1 apis/rsc/rcachev1 apis/rsc/rratelimitv1
	mkdir -p apis/rsc/renterprisev1 apis/rsc/raccessv1
	protoc -I . -I $(PROTO_IN_RSC)/metav1 rmetav1.proto \
		--go_out=apis/rsc/rmetav1 --go-grpc_out=apis/rsc/rmetav1 $(PROTO_GO_OPT)
	protoc -I . -I $(PROTO_IN_RSC)/corev1 rcorev1.proto \
		--go_out=apis/rsc/rcorev1 --go-grpc_out=apis/rsc/rcorev1 $(PROTO_GO_OPT_GRPC)

	protoc -I . -I $(PROTO_IN_RSC)/enterprisev1 renterprisev1.proto \
		--go_out=apis/rsc/renterprisev1 --go-grpc_out=apis/rsc/renterprisev1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_RSC)/cachev1 rcachev1.proto \
		--go_out=apis/rsc/rcachev1 --go-grpc_out=apis/rsc/rcachev1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_RSC)/accessv1 raccessv1.proto \
		--go_out=apis/rsc/raccessv1 --go-grpc_out=apis/rsc/raccessv1 $(PROTO_GO_OPT_GRPC)
	protoc -I . -I $(PROTO_IN_RSC)/ratelimitv1 rratelimitv1.proto \
		--go_out=apis/rsc/rratelimitv1 --go-grpc_out=apis/rsc/rratelimitv1 $(PROTO_GO_OPT_GRPC)

gen-go-client:
	mkdir -p apis/client/cliconfigv1
	protoc -I . -I $(PROTO_IN_CLIENT)/configv1 configv1.proto \
		--go_out=apis/client/cliconfigv1 --go-grpc_out=apis/client/cliconfigv1 $(PROTO_GO_OPT)

cp-pb:
	cp -r ../pb/apis/protobuf ./apis

gen-api-console:
	cd ./cluster/console/console/web/package; npm run protoc

gen-api: cp-pb gen-go-main gen-go-cluster gen-go-client gen-go-rsc gen-api-console gen-json-schema
	rm -rf ./apis/protobuf
	go run unsorted/licenser/main.go

gen-json-schema:
	go install github.com/chrusty/protoc-gen-jsonschema/cmd/protoc-gen-jsonschema@latest
	mkdir -p tmp/jsonschema/core
	mkdir -p tmp/jsonschema/enterprise
	protoc -I . -I $(PROTO_IN_MAIN)/corev1 \
		--jsonschema_opt=enums_as_strings_only \
		--jsonschema_opt=disallow_bigints_as_strings \
		--jsonschema_opt=disallow_additional_properties \
		--jsonschema_opt=enforce_oneof \
		--jsonschema_out=./tmp/jsonschema/core --proto_path=$(PROTO_IN_MAIN)/corev1 corev1.proto
	protoc -I . -I $(PROTO_IN_MAIN)/corev1 -I $(PROTO_IN_MAIN)/enterprisev1 \
		--jsonschema_opt=enums_as_strings_only \
		--jsonschema_opt=disallow_bigints_as_strings \
		--jsonschema_opt=disallow_additional_properties \
		--jsonschema_opt=enforce_oneof \
		--jsonschema_out=./tmp/jsonschema/enterprise --proto_path=$(PROTO_IN_MAIN)/enterprisev1 enterprisev1.proto
	cp -r ./tmp/jsonschema/core ./cluster/console/console/web/package/src/jsonschema
	cp -r ./tmp/jsonschema/enterprise ./cluster/console/console/web/package/src/jsonschema


tidy:
	cd apis; $(CMD_TIDY)
	cd pkg; $(CMD_TIDY)
	cd cluster/common; $(CMD_TIDY)
	cd cluster/rscserver; $(CMD_TIDY)
	cd cluster/apiserver; $(CMD_TIDY)
	cd cluster/dirsync; $(CMD_TIDY)
	cd cluster/secretman; $(CMD_TIDY)
	cd cluster/genesis; $(CMD_TIDY)
	cd cluster/nocturne; $(CMD_TIDY)
	cd cluster/cloudman; $(CMD_TIDY)
	cd cluster/collector; $(CMD_TIDY)
	cd cluster/console; $(CMD_TIDY)
	cd cluster/clusterman; $(CMD_TIDY)
	cd cluster/mockapiserver; $(CMD_TIDY)
	cd cluster/publicserver; $(CMD_TIDY)
	cd cluster/logstore; $(CMD_TIDY)
	cd cluster/rscstore; $(CMD_TIDY)
	cd cluster/metricstore; $(CMD_TIDY)
	cd cluster/policyportal; $(CMD_TIDY)
	cd cluster/e2e; $(CMD_TIDY)