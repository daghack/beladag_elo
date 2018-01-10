FROM golang:1.9 as compiler
WORKDIR /go/src/app
COPY vendor ./vendor
COPY src ./src
RUN go get github.com/constabulary/gb/...
RUN gb vendor restore
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 gb build

FROM debian:latest
RUN mkdir /app
WORKDIR /app
COPY --from=compiler /go/src/app/bin/web-linux-amd64 ./web
COPY templates templates
ENTRYPOINT ["/app/web"]
