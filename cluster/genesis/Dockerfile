FROM golang:1.25.8 AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN make build-genesis
FROM alpine:3.23
RUN apk --no-cache add ca-certificates bash curl openssl
RUN adduser -S -D -h /app octelium
# RUN chown octelium /app
USER octelium
COPY --from=builder /build/bin/octeliumee-genesis /app/
ENTRYPOINT ["/app/octeliumee-genesis"]
