.PHONY: ops build run test demo replicate

ops:
	docker-compose up

build:
	docker-compose run --rm --name content_service_golang golang go build

run:
	docker-compose run --rm --name content_service_golang -p 8000:8000 golang /bin/bash -c "go build && ./content-service"

test:
	docker-compose run --rm --name content_service_golang golang go test -v

demo:
	docker-compose run --rm --name content_service_golang -p 8000:8000 \
		-e AWS_REGION=$(AWS_REGION) -e AWS_ACCESS_KEY=$(AWS_ACCESS_KEY) -e AWS_SECRET_KEY=$(AWS_SECRET_KEY) \
		golang /bin/bash -c "go build && ./content-service --s3"

replicate:
	docker-compose run --rm --name content_service_replicate -p 8001:8000 golang /bin/bash -c "cd cmd/replication && go build && ./replication"
