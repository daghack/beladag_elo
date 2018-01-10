FROM golang:1.9 as compiler
RUN go get github.com/constabulary/gb/...
WORKDIR /go/src/app
COPY vendor ./vendor
RUN gb vendor restore
COPY src ./src
RUN gb build

FROM debian:latest
RUN mkdir /app
WORKDIR /app
COPY --from=compiler /go/src/app/bin/web ./web
COPY templates ./templates
CMD ["./web"]
