# smtprelay

[![Go Report Card](https://goreportcard.com/badge/github.com/decke/smtprelay)](https://goreportcard.com/report/github.com/decke/smtprelay)

Simple Golang based SMTP relay/proxy server that accepts mail via SMTP
and forwards it directly to another SMTP server.


## Why another SMTP server?

Outgoing mails are usually send via SMTP to an MTA (Mail Transfer Agent)
which is one of Postfix, Exim, Sendmail or OpenSMTPD on UNIX/Linux in most
cases. You really don't want to setup and maintain any of those full blown
kitchensinks yourself because they are complex, fragile and hard to
configure.

My use case is simple. I need to send automatically generated mails from
cron via msmtp/sSMTP/dma, mails from various services and network printers
to GMail without giving away my GMail credentials to each device which
produces mail.


## Main features

* Supports SMTPS/TLS (465), STARTTLS (587) and unencrypted SMTP (25)
* Checks for sender, receiver, client IP
* Authentication support with file (LOGIN, PLAIN)
* Enforce encryption for authentication
* Forwards all mail to a smarthost (GMail, MailGun or any other SMTP server)
* Small codebase
* IPv6 support

## Development
- `make build` - build go code
- `make docker` build docker image
- `make docker-push` - build and push docker image

Update `DOCKER_TAG` in `config.mk` to change Docker Repo.

note: tags for docker images are generated by `scripts/version`

## Deployment

### Configuration
There are two ways to provide configuration

1. .ini file (see `smtprelay.ini` for example)
```bash
./smtprelay -config smtprelay.ini
```

2. command line arguments for each config option.
```bash
./smtprelay -listen=127.0.0.1:2525 -hostname=localhost -remote_host=smtp.mailgun.org:2525 -remote_user=hosted-grafana@grafana.net
```

You can mix and match, see priority to see which config value will be used

**config priority** - [source](https://github.com/vharitonsky/iniflags/#hybrid-configuration-library)
1. use value set via command-line,
2. if not set, use value from ini file,
3. at last, use default value.

NOTE: If `remote_pass` is not set at the end, It will try to read
it from `REMOTE_PASS` environment variable.

It was added to support loading secret from environment variable

Use `./smtprelay -help` for help on config options.

### Metrics

Service exposes Prometheus metrics on `<url>:8080/metrics`, metrics port
is configurable using `metrics_listen`

### Logs

Structured logs are written to `stdout` by default, can be configured to write to file by
setting `logfile` to full path of logfile.

log level is `debug` by default, can be changed by setting `log_level`.

supported levels: `trace, debug, info, warning, error, fatal, panic`

### Docker

We publish images on DockerHub at `grafana/smtprelay`

See [all tags](https://hub.docker.com/r/grafana/smtprelay/tags)

### Testing
To test code or config, start smtprelay, and send test email using `swaks`.

> Tip: you can install `swaks` using `sudo apt install swaks` on Ubuntu.

```bash
swaks --to <email> --from=<email> --server localhost:2525 --h-Subject: "Hello from smtprelay" --h-Body: "This is test email from smtprelay"
```
