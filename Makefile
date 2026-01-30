.PHONY: all swagger docker install test bench

SWAG ?= ~/go/bin/swag

GO_ENV += CGO_ENABLED=0
TARGET := $(shell find . -type f -name '*.go')

all: swagger docker install

docker: judge
	docker compose up -d --force-recreate --build backend worker

install:
	$(GO_ENV) go install

test:
	go test -v -count=1 -failfast ./tests/units
	go test -v -count=1 -failfast ./tests/e2e

bench:
	go test -bench=. ./tests/benchmarks

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmd/serve/backend.go -pdl 1

judge: $(TARGET)
	$(GO_ENV) go build -o judge
