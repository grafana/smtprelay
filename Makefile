
ROOTDIR := $(abspath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
DISTDIR := $(abspath $(ROOTDIR)/dist)

BUILD_VERSION := $(shell $(ROOTDIR)/scripts/version)
BUILD_COMMIT := $(shell git rev-parse HEAD^{commit})
BUILD_STAMP := $(shell date --utc --rfc-3339=seconds)

include config.mk

-include local/Makefile

build:
	go build -v .

clean:
	rm smtprelay

.PHONY: docker
docker: build
	docker build \
		--build-arg=GIT_REVISION=$(BUILD_COMMIT) \
		-t $(DOCKER_TAG) \
		.

.PHONY: docker-push
docker-push:  docker
	docker push $(DOCKER_TAG)
	docker tag $(DOCKER_TAG) $(DOCKER_TAG):$(BUILD_VERSION)
	docker push $(DOCKER_TAG):$(BUILD_VERSION)

.PHONY: lint
lint:
	@golangci-lint run --max-same-issues=0 --max-issues-per-linter=0 -v
