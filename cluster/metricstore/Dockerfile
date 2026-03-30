FROM golang:1.25.8-bookworm AS builder
RUN apt-get update && apt-get install -y git g++ cmake ninja-build libssl-dev
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN make build-metricstore

FROM debian:12
RUN groupadd -r octelium && useradd -r -m -g octelium octelium
WORKDIR /app
USER octelium
COPY --from=builder /build/bin/octeliumee-metricstore /app/
ENTRYPOINT ["/app/octeliumee-metricstore"]
