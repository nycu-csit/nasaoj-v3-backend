.PHONY: all \
	docker docker-quiet docker-clean docker-init docker-reset \
	install swagger test bench redis postgres

SWAG ?= go run github.com/swaggo/swag/cmd/swag@v1.16.6


DOCKER ?= docker
DOCKER_BUILD_ARGS +=
COMPOSE_ARGS += --progress plain
COMPOSE_ENV +=
COMPOSE ?= $(COMPOSE_ENV) $(DOCKER) compose $(COMPOSE_ARGS)
export COMPOSE

GO_ENV += CGO_ENABLED=0
SOURCE := $(shell find . -type f -name '*.go')
TARGET := judge

all: swagger docker install

docker-reset: docker-clean docker docker-init
	sleep 1

docker: judge
	$(COMPOSE) up -d --wait \
		--force-recreate \
		--build backend worker

docker-quiet: judge
	$(COMPOSE) up -d --wait \
		--force-recreate \
		--build backend worker \
		--quiet-build --quiet-pull

docker-clean:
	$(COMPOSE) down -v --remove-orphans

docker-init:
	./docker/init.sh

postgres:
	$(COMPOSE) exec postgres psql \
		postgres://nasaoj:nasaoj@postgres:5432/nasaoj?sslmode=disable

redis:
	$(COMPOSE) exec redis redis-cli

install:
	$(GO_ENV) go install

test: install
	go test -v -failfast ./tests/units -count=1
	go test -v -failfast ./tests/e2e -count=1

bench:
	go test -bench=. ./tests/benchmarks

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmds/serve/backend.go -pdl 1

$(TARGET): $(SOURCE)
	$(GO_ENV) go build -o $@
