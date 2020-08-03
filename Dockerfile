FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/websu-io/websu
COPY . .

RUN go get -d -v ./...
RUN go build ./cmd/websu-api && mv websu-api /

CMD ["app"]

FROM justinribeiro/lighthouse

COPY --from=builder /websu-api /websu-api

ENTRYPOINT ["/websu-api"]

EXPOSE 8000/tcp
