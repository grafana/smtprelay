FROM golang:1.21-alpine as build

RUN apk add --no-cache ca-certificates make git

WORKDIR /go/src/github.com/grafana/smtprelay

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ENV GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT}

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download -x

COPY . ./
RUN make build

FROM alpine:3.19 AS runtime

RUN apk add --no-cache --upgrade \
        libcrypto3>=3.1.4-r1 \
        libssl3>=3.1.4-r1

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/grafana/smtprelay/smtprelay /usr/local/bin/smtprelay

ARG GIT_REVISION

LABEL org.opencontainers.image.revision=$GIT_REVISION \
        org.opencontainers.image.vendor="Grafana Labs" \
        org.opencontainers.image.title="smtprelay" \
        org.opencontainers.image.source="https://github.com/grafana/smtprelay"

# users need to mount config file at /usr/local/smtprelay.ini
ENTRYPOINT [ "/usr/local/bin/smtprelay" ]
CMD [ "--help" ]
