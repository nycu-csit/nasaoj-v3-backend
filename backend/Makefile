.PHONY: all swagger run

all: swagger backend

swagger:
	~/go/bin/swag fmt
	~/go/bin/swag init -g backend.go -pdl 1

backend: *.go handlers/*.go middlewares/*.go ../models/*.go
	go mod tidy
	go build

run: 
	go run .
