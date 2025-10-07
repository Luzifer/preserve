FROM golang:1.25-alpine@sha256:352f1ef84634653587abc7f3bdf920eb2a96887a430c23e202a600239eaeedf7 AS builder

COPY . /src/preserve
WORKDIR /src/preserve

RUN set -ex \
 && apk add --no-cache \
      git \
 && go install \
      -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -trimpath


FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1

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
