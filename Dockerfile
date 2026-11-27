ARG ALPINE_VERSION=3.24

FROM golang:1.26-alpine${ALPINE_VERSION} AS build-env

ARG ADDITIONAL_BUILD_TAGS=""

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk add --no-cache build-base

WORKDIR /maddy

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN mkdir -p /pkg/data && \
    cp maddy.conf.docker /pkg/data/maddy.conf && \
    ./build.sh --builddir /tmp --destdir /pkg/ --tags "docker ${ADDITIONAL_BUILD_TAGS}" build install

FROM alpine:${ALPINE_VERSION}
LABEL maintainer="fox.cpp@disroot.org"
LABEL org.opencontainers.image.source=https://github.com/foxcpp/maddy

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk --no-cache add ca-certificates bash tini
COPY --from=build-env /pkg/data/maddy.conf /data/maddy.conf
COPY --from=build-env /pkg/usr/local/bin/maddy /bin/

EXPOSE 25 143 993 587 465
VOLUME ["/data"]
ENTRYPOINT ["tini", "--", "/bin/maddy", "-config", "/data/maddy.conf"]
CMD ["run"]
