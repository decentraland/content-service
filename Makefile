VERSION := $(shell git rev-list -1 HEAD)
BUILD_FLAGS = -ldflags '-X github.com/decentraland/dcl-gin/pkg/dclgin.version=$(VERSION)'

build:
	go build $(BUILD_FLAGS) -o build/content ./cmd/service

init:
	git config core.hooksPath .githooks

test:
	go test -v ./... -count=1

run:
	make build
	AWS_REGION=$(AWS_REGION) AWS_ACCESS_KEY=$(AWS_ACCESS_KEY) AWS_SECRET_KEY=$(AWS_SECRET_KEY) ./build/content

dev-env:
	docker-compose up

integration:
	docker start cs_localstack \
        && aws --endpoint-url=http://localhost:4572 s3 mb s3://local-content \
        && aws --endpoint-url=http://localhost:4572 s3 mb s3://local-mappings \
        && AWS_REGION="us-east-1" AWS_ACCESS_KEY="something" AWS_SECRET_KEY="something" /bin/bash -c 'go test -count=1 -tags=integration ./test/integration/integration_test.go' \
        && docker stop cs_localstack

.PHONY: build