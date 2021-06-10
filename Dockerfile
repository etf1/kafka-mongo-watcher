FROM golang:1.16-alpine3.13 as builder

WORKDIR /project
COPY . .

RUN apk --no-cache add gcc libc-dev

RUN GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o kafka-mongo-watcher ./cmd/watcher/

FROM alpine:3.13
LABEL name="kafka-mongo-watcher"

WORKDIR /
COPY --from=builder /project/kafka-mongo-watcher kafka-mongo-watcher

CMD ["./kafka-mongo-watcher"]
