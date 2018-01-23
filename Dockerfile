FROM golang:1.9 as compiler
RUN go get github.com/constabulary/gb/...
WORKDIR /go/src/app
COPY vendor ./vendor
COPY src ./src
RUN gb vendor restore
RUN gb build

FROM debian:latest
RUN apt-get update && apt-get install -y ca-certificates
RUN mkdir /app
WORKDIR /app
COPY --from=compiler /go/src/app/bin/web ./web
COPY templates ./templates
CMD ["./web"]
