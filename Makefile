.DEFAULT_GOAL = build

ROOTDIR := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
BIN_DIR := $(ROOTDIR)/bin

BUILD_VERSION := $(shell $(ROOTDIR)/scripts/version)
BUILD_COMMIT := $(shell git rev-parse HEAD^{commit})
DOCKER_TAG ?= grafana/smtprelay

$(BIN_DIR)/smtprelay: $(shell find . -type f -name '*.go') go.mod go.sum
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 \
		go build \
			-trimpath \
			-o $@ \
			.

build: $(BIN_DIR)/smtprelay

clean:
	@rm -rf $(BIN_DIR)
	@rm -rf *.out

.PHONY: test
test:
	go test -race -coverprofile=c.out ./...

.PHONY: docker
docker:
	docker build \
		--build-arg=GIT_REVISION=$(BUILD_COMMIT) \
		-t $(DOCKER_TAG) \
		.

.PHONY: docker-push
docker-push: docker
	docker push $(DOCKER_TAG)
	docker tag $(DOCKER_TAG) $(DOCKER_TAG):$(BUILD_VERSION)
	docker push $(DOCKER_TAG):$(BUILD_VERSION)

.PHONY: lint
lint:
	@golangci-lint run --max-same-issues=0 --max-issues-per-linter=0 -v
