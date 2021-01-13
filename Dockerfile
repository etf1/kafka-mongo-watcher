FROM golang:1.15-alpine3.12 as builder

WORKDIR /project
COPY . .

RUN apk --no-cache add gcc libc-dev

RUN GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o kafka-mongo-watcher ./cmd/watcher/

FROM alpine:3.12
LABEL name="kafka-mongo-watcher"

WORKDIR /
COPY --from=builder /project/kafka-mongo-watcher kafka-mongo-watcher

CMD ["./kafka-mongo-watcher"]
