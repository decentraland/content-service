.PHONY: ops build run test integration demo replicate

COMMIT := $(shell git rev-list -1 HEAD)

ops:
	docker-compose up

init:
	git config core.hooksPath .githooks

build:
	docker-compose run --rm --name content_service_golang golang go build -ldflags '-X github.com/decentraland/dcl-gin/pkg/dclgin.version=$(COMMIT)'

run:
	docker-compose run --rm --name content_service_golang -p 8000:8000 golang /bin/bash -c "go build -ldflags '-X github.com/decentraland/dcl-gin/pkg/dclgin.version=$(COMMIT)' && ./content-service"

test:
	go test -v ./... -count=1

integration:
	docker start content_service_redis \
        && AWS_REGION=$(AWS_REGION) AWS_ACCESS_KEY=$(AWS_ACCESS_KEY) AWS_SECRET_KEY=$(AWS_SECRET_KEY) RUN_IT=true /bin/bash -c 'go test -v main.go integration_test.go -count=1' \
        && docker stop content_service_redis


replicate:
	docker-compose run --rm --name content_service_replicate -p 8001:8000 golang /bin/bash -c "go build cmd/replication/replication.go && ./replication"
