.PHONY: all swagger docker install

SWAG ?= ~/go/bin/swag

TARGET := $(shell find . -type f -name '*.go')

all: docker install

docker: swagger judge
	docker compose up -d --force-recreate --build backend worker

install: judge
	go install

swagger:
	$(SWAG) fmt
	$(SWAG) init -o backend/docs -g cmd/serve/backend.go -pdl 1

judge: $(TARGET)
	CGO_ENABLED=0 go build -o judge
