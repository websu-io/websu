FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/reviewor-org/speedster
COPY . .

RUN go get -d -v ./...
RUN go build ./cmd/speedster-api && mv speedster-api /

CMD ["app"]

FROM justinribeiro/lighthouse

COPY --from=builder /speedster-api /speedster-api

ENTRYPOINT ["/speedster-api"]

EXPOSE 8000/tcp
