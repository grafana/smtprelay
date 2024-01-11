# smtprelay

This is a simple SMTP relay, that accepts mail via SMTP and forwards it to
another SMTP server. Its main purpose is focused on observability, with metrics
exposed in the Prometheus format, and structured logging.

Grafana Labs uses this internally, and as such it is not intended to be a
generally-supported product. In particular, we reserve the right to make
breaking changes, remove functionality, or otherwise change the code without
notice.

We are happy to consider issues and pull requests.

## Development
- `make build` - build go code
- `make docker` build docker image
- `make docker-tag` - build and push docker image
- `make docker-push` - build, tag, and push docker image

Update `DOCKER_IMAGE` in `Makefile` to change Docker Repo.

## Deployment

### Configuration
There are two ways to provide configuration

1. .ini file (see `smtprelay.ini` for example)
    ```console
    $ ./smtprelay -config=smtprelay.ini
    ```

2. command line arguments for each config option.
    ```console
    $ ./smtprelay -listen=127.0.0.1:2525 -hostname=localhost -remote_host=smtp.example.com:587 -remote_user=noreply@example.com
    ```

You can mix and match, see priority to see which config value will be used

**config priority** - [source](https://github.com/vharitonsky/iniflags/#hybrid-configuration-library)
1. use value set via command-line,
2. if not set, use value from ini file,
3. at last, use default value.

NOTE: If `remote_pass` is not set at the end, It will try to read
it from `REMOTE_PASS` environment variable.

It was added to support loading the secret from an environment variable.

Use `./smtprelay -help` for help on config options.

### Metrics

Prometheus metrics are available at `<url>:8080/metrics`.

The listening address can be changed by setting `metrics_listen`.

### Logs

Structured logs are written to `stderr`.

The log level is `INFO` by default, and can be changed by setting `log_level`.

### Docker

We publish images on DockerHub at [`grafana/smtprelay`](https://hub.docker.com/r/grafana/smtprelay)

### Manual Testing

To test code or config, start smtprelay, and send test email using `swaks`.

> Tip: you can install `swaks` using `sudo apt install swaks` on Ubuntu.

```console
$ swaks --to=test@example.com --from=noreply@example.com --server=localhost:2525 --h-Subject="Hello from smtprelay" --body="This is test email from smtprelay"
```

### Acknowledgements

This started as a fork of [github.com/decke/smtprelay](https://github.com/decke/smtprelay).
We thank the original authors for their work.
