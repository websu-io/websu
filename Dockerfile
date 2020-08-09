FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/websu-io/websu
COPY go.mod go.sum ./
# Download dependencies and cache in docker layer
RUN go mod download
COPY . .
RUN go build ./cmd/websu-api && mv websu-api /

FROM justinribeiro/lighthouse

WORKDIR /home/chrome
COPY --from=builder /websu-api /home/chrome/websu-api
RUN mkdir static && \
    wget https://github.com/websu-io/websu-ui/releases/latest/download/build.tar.gz && \
    tar -xzf build.tar.gz -C static && \
    rm build.tar.gz

ENTRYPOINT ["/home/chrome/websu-api"]

EXPOSE 8000/tcp
