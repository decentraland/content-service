.PHONY: run

run:
	docker-compose up

build:
	go build

test:
	go test