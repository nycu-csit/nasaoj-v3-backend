.PHONY: all swagger

all: swagger

swagger:
	~/go/bin/swag fmt
	~/go/bin/swag init -o backend/docs -g cmd/serve/backend.go -pdl 1
