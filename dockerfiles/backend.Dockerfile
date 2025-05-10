FROM dockersource.cc.cs.nctu.edu.tw/devops/images/golang/1.24.3:main AS build

RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /go/src/app
COPY . .

WORKDIR /go/src/app/backend
RUN make deploy

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /go/bin/app /
COPY ./backend/ /
CMD ["/app"]
