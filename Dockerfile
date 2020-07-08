ARG IMAGE_BASE=golang
ARG GO_VERSION=1.14

ARG VERSION
ARG COMMIT
ARG DATE

FROM ${IMAGE_BASE}:${GO_VERSION} AS builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on

ADD . ./app

WORKDIR app

RUN go mod vendor

RUN go build \
    -a -mod=vendor \
    -ldflags="-s -w \
    -X main.version=${VERSION} \
    -X main.commit=${COMMIT} \
    -X main.date=${DATE} \
    -X main.builtBy=Docker" \
    -o ./gaos

ENTRYPOINT ["./gaos"]

