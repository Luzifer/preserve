FROM golang:1.26.4-alpine@sha256:7a3e50096189ad57c9f9f865e7e4aa8585ed1585248513dc5cda498e2f41812c AS builder

COPY . /src/preserve
WORKDIR /src/preserve

RUN set -ex \
 && apk add --no-cache \
      git \
 && go install \
      -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -trimpath


FROM alpine:3.24@sha256:a2d49ea686c2adfe3c992e47dc3b5e7fa6e6b5055609400dc2acaeb241c829f4

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
