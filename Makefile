.PHONY: all swagger docker

SWAG ?= ~/go/bin/swag

TARGET := $(shell find . -type f -name '*.go')

all: judge

docker: judge
	docker compose up -d --force-recreate --build backend worker

judge: swagger $(TARGET)
	CGO_ENABLED=0 go build -o judge

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmd/serve/backend.go -pdl 1
