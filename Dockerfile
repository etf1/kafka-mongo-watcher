FROM golang:1.23-alpine as builder

ARG VERSION

WORKDIR /project
COPY . .

RUN apk --no-cache add gcc libc-dev

RUN GOOS=linux GOARCH=amd64 go build -tags 'musl' -ldflags "-s -w -X github.com/etf1/kafka-mongo-watcher/config.AppVersion=$VERSION" -o kafka-mongo-watcher ./cmd/watcher/

FROM alpine:3.21
LABEL name="kafka-mongo-watcher"

WORKDIR /
COPY --from=builder /project/kafka-mongo-watcher kafka-mongo-watcher
COPY --from=builder /project/public /public

CMD ["./kafka-mongo-watcher"]
