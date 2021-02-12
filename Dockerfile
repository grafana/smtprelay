FROM debian:stable-slim

RUN apt-get update && apt-get -y install ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY smtprelay /usr/local/bin/smtprelay
# users need to mount config file at /usr/local/smtprelay.ini
ENTRYPOINT ["/usr/local/bin/smtprelay"]
