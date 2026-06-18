FROM golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

COPY . /src/preserve
WORKDIR /src/preserve

RUN set -ex \
 && apk add --no-cache \
      git \
 && go install \
      -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -trimpath


FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

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
