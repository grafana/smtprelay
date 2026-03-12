FROM golang:1.26-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS build

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

FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659 AS runtime

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
