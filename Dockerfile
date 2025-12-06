FROM golang:1.25-alpine@sha256:26111811bc967321e7b6f852e914d14bede324cd1accb7f81811929a6a57fea9 AS builder

COPY . /src/preserve
WORKDIR /src/preserve

RUN set -ex \
 && apk add --no-cache \
      git \
 && go install \
      -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -trimpath


FROM alpine:3.23@sha256:51183f2cfa6320055da30872f211093f9ff1d3cf06f39a0bdb212314c5dc7375

LABEL maintainer="Knut Ahlers <knut@ahlers.me>"

ENV STORAGE_DIR=/data

RUN set -ex \
 && apk --no-cache add \
      ca-certificates

COPY --from=builder /go/bin/preserve /usr/local/bin/preserve

EXPOSE 3000
VOLUME ["/data"]

USER 1000

ENTRYPOINT ["/usr/local/bin/preserve"]
CMD ["--"]

# vim: set ft=Dockerfile:
