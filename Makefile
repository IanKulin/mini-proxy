# Makefile

# Default target if you just type `make`
.PHONY: run
run:
	TARGET_HEALTH_URL=https://api.ato.gov.au/healthcheck/v1/ RATE_LIMIT_WHITELIST_IP=127.0.0.1 LOG_LEVEL=debug go run main.go

.PHONY: build
build:
	go build -o mini-proxy main.go

.PHONY: docker-build
docker-build:
	docker build -t ghcr.io/iankulin/mini-proxy .

.PHONY: docker-push
docker-push: docker-build
	docker push ghcr.io/iankulin/mini-proxy

.PHONY: test
test:
	go test -v ./...
