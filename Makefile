.PHONY: ops build run test integration demo replicate

VERSION := $(shell git rev-list -1 HEAD)
BUILD_FLAGS = -ldflags '-X github.com/decentraland/dcl-gin/pkg/dclgin.version=$(VERSION)'

build:
	go build $(BUILD_FLAGS) -o build/content ./cmd/service

init:
	git config core.hooksPath .githooks

test:
	go test -v ./... -count=1

dev-env:
	docker-compose up

integration:
	docker start content_service_redis \
        && AWS_REGION=$(AWS_REGION) AWS_ACCESS_KEY=$(AWS_ACCESS_KEY) AWS_SECRET_KEY=$(AWS_SECRET_KEY) RUN_IT=true /bin/bash -c 'go test -v main.go integration_test.go -count=1' \
        && docker stop content_service_redis
