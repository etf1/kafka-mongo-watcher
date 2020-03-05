FROM golang:1.14-alpine as builder

WORKDIR /project
COPY . .

RUN apk --no-cache update && \
    apk --no-cache add gcc libc-dev librdkafka-dev pkgconfig

RUN GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o kafka-mongo-watcher ./cmd/watcher/

FROM alpine:3.11
LABEL name="kafka-mongo-watcher"

RUN apk --no-cache update && \
    apk --no-cache add librdkafka

WORKDIR /
COPY --from=builder /project/kafka-mongo-watcher kafka-mongo-watcher

CMD ["./kafka-mongo-watcher"]