.PHONY: ops build run test

ops:
	docker-compose up

build:
	docker-compose run --rm --name content_service_golang golang go build

run:
	docker-compose run --rm --name content_service_golang -p 8000:8000 golang /bin/bash -c "go build && ./content-service"

test:
	docker-compose run --rm --name content_service_golang golang go test
