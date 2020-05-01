FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/reviewor-org/speedster
COPY . .

RUN go get -d -v ./...
RUN go build -o /speedster

CMD ["app"]

FROM justinribeiro/lighthouse

COPY --from=builder /speedster /speedster

ENTRYPOINT ["/speedster"]

EXPOSE 8000/tcp
