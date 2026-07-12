FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

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
