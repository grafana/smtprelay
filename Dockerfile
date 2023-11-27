FROM golang:1.18-stretch as build
ARG GOARCH="amd64"
COPY . /build_dir
WORKDIR /build_dir
RUN make clean && make build

FROM debian:stable-slim
RUN apt-get update && apt-get -y install ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build /build_dir/smtprelay /usr/local/bin/smtprelay
# users need to mount config file at /usr/local/smtprelay.ini
ENTRYPOINT ["/usr/local/bin/smtprelay"]
