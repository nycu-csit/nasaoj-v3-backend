.PHONY: all swagger docker install test

SWAG ?= ~/go/bin/swag

TARGET := $(shell find . -type f -name '*.go')

all: swagger docker install

docker: judge
	docker compose up -d --force-recreate --build backend worker

install: judge
	go install

test:
	go test ./...

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmd/serve/backend.go -pdl 1

judge: $(TARGET)
	CGO_ENABLED=0 go build -o judge
