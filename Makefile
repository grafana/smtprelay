.DEFAULT_GOAL = build

BIN_DIR := bin

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
VERSION := $(shell go run ./scripts/version.go)
GO_LDFLAGS ?= $(shell go run ./scripts/version.go -g)
GO_FLAGS   := -trimpath -ldflags "$(GO_LDFLAGS)"
GIT_REVISION ?= $(shell git rev-parse --short HEAD)
DOCKER_IMAGE ?= grafana/smtprelay

$(BIN_DIR)/smtprelay: $(shell find . -type f -name '*.go') go.mod go.sum
	CGO_ENABLED=0 \
		go build \
			$(GO_FLAGS) \
			-o $@ \
			.

build: $(BIN_DIR)/smtprelay

clean:
	@rm -rf $(BIN_DIR)
	@rm -rf *.out
	@rm -rf smtprelay.version

.PHONY: test
test:
	go test -race -coverprofile=c.out ./...

.PHONY: docker
docker:
	docker build \
		--build-arg=VERSION=$(VERSION) \
		--build-arg=GIT_REVISION=$(GIT_REVISION) \
		--build-arg=GO_LDFLAGS='$(GO_LDFLAGS)' \
		-t $(DOCKER_IMAGE) \
		.

.PHONY: docker-tag
docker-tag: smtprelay.version

smtprelay.version: docker
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(VERSION)
	echo "$(VERSION)" > $@

.PHONY: docker-push
docker-push: docker-tag
	docker push $(DOCKER_IMAGE)
	docker push $(DOCKER_IMAGE):$(VERSION)

.PHONY: lint
lint:
	@golangci-lint run --max-same-issues=0 --max-issues-per-linter=0 -v

.PHONY: release
release:
	@go run ./scripts/version.go -release
