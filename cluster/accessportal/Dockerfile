FROM node:24 AS builder-web
COPY cluster/accessportal/accessportal/web/package /build
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN cd /build && npm ci && npm run build

FROM golang:1.26.4 AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
COPY --from=builder-web /build/dist /build/cluster/accessportal/accessportal/web
RUN make build-accessportal
FROM alpine:3.23
RUN apk --no-cache add ca-certificates
RUN adduser -S -D -H -u 3454 -h /app octelium
USER octelium
COPY --from=builder /build/bin/octeliumee-accessportal /app/
ENTRYPOINT ["/app/octeliumee-accessportal"]
