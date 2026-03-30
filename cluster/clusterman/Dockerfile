
FROM golang:1.25.8 AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN make build-clusterman
FROM alpine:3.23
RUN apk --no-cache add ca-certificates bash curl openssl
RUN curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
RUN adduser -S -D -H -u 3454 -h /app octelium
USER octelium
COPY --from=builder /build/bin/octeliumee-clusterman /app/
ENTRYPOINT ["/app/octeliumee-clusterman"]
CMD ["run"]
