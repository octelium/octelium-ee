FROM golang:1.26.4-bookworm AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        git \
        g++ \
        cmake \
        ninja-build \
        libssl-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

ADD . /build/

RUN make build-metricstore

FROM debian:12

ARG OCTELIUM_UID=3454
ARG OCTELIUM_GID=3454

RUN groupadd --gid "${OCTELIUM_GID}" octelium \
    && useradd \
        --uid "${OCTELIUM_UID}" \
        --gid "${OCTELIUM_GID}" \
        --home-dir /app \
        --no-create-home \
        --shell /usr/sbin/nologin \
        octelium \
    && mkdir -p /app /octelium-data \
    && chown -R "${OCTELIUM_UID}:${OCTELIUM_GID}" /app /octelium-data

WORKDIR /app

COPY --from=builder --chown=${OCTELIUM_UID}:${OCTELIUM_GID} /build/bin/octeliumee-metricstore /app/

USER ${OCTELIUM_UID}:${OCTELIUM_GID}

ENTRYPOINT ["/app/octeliumee-metricstore"]