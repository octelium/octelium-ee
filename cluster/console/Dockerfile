FROM node:24 AS builder-web
COPY cluster/console/console/web/package /build
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN cd /build && npm ci && npm run build

FROM golang:1.25.8 AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
COPY --from=builder-web /build/dist /build/cluster/console/console/web
RUN make build-console
FROM alpine:3.23
RUN apk --no-cache add ca-certificates
RUN adduser -S -D -H -u 3454 -h /app octelium
USER octelium
COPY --from=builder /build/bin/octeliumee-console /app/
ENTRYPOINT ["/app/octeliumee-console"]
