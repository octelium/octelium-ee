
FROM golang:1.25.8 AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN make build-dirsync
FROM alpine:3.23
RUN apk --no-cache add ca-certificates
RUN adduser -S -D -H -u 3454 -h /app octelium
USER octelium
EXPOSE 8080
COPY --from=builder /build/bin/octeliumee-dirsync /app/
ENTRYPOINT ["/app/octeliumee-dirsync"]

