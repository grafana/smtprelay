FROM golang:1.24-alpine@sha256:7772cb5322baa875edd74705556d08f0eeca7b9c4b5367754ce3f2f00041ccee as build

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
# sanity check - make sure the binary runs and is executable
RUN bin/smtprelay --version

FROM alpine:3.21@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c AS runtime

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/grafana/smtprelay/bin/smtprelay /usr/local/bin/smtprelay

ARG GIT_REVISION

LABEL org.opencontainers.image.revision=$GIT_REVISION \
        org.opencontainers.image.vendor="Grafana Labs" \
        org.opencontainers.image.title="smtprelay" \
        org.opencontainers.image.source="https://github.com/grafana/smtprelay"

# users need to mount config file at /usr/local/smtprelay.ini
ENTRYPOINT [ "/usr/local/bin/smtprelay" ]
CMD [ "--help" ]
