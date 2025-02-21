FROM golang:1.24 AS build

WORKDIR /go/src/app
COPY . .

WORKDIR /go/src/app/scheduler
RUN go mod download
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /go/bin/app /
COPY ./scheduler/ /
CMD ["/app"]
