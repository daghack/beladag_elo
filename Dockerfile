FROM golang:1.9 as compiler
WORKDIR /go/src/app
COPY vendor ./vendor
COPY src ./src
RUN go get github.com/constabulary/gb/...
RUN gb vendor restore
RUN gb build

FROM debian:latest
RUN mkdir /app
WORKDIR /app
COPY --from=compiler /go/src/app/bin/web ./web
COPY templates ./templates
CMD ["./web"]
