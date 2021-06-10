# Kafka MongoDB Watcher

![Test (master)](https://github.com/etf1/kafka-mongo-watcher/workflows/Test%20(master)/badge.svg)
[![GoDoc](https://godoc.org/github.com/etf1/kafka-mongo-watcher?status.png)](https://godoc.org/github.com/etf1/kafka-mongo-watcher)

This project listens for a MongoDB collection events (insert, update, delete, ...) also called "oplogs" for operation logs and distribute them into a Kafka topic of your choice.
There is also a replay mode that allows you to initialize all items of a collection into a Kafka topic for the first time.

## Prerequisites

In addition of the binary, you will also need the following the Kafka library:

* [librdkafka](https://github.com/edenhill/librdkafka) (report to the Installation part)

## Installation

### Download binary

You can download the latest version of the binary built for your architecture here:

* Architecture **i386** [
    [Darwin](https://github.com/etf1/kafka-mongo-watcher/releases/latest/download/kafka-mongo-watcher-darwin-386) /
    [Linux](https://github.com/etf1/kafka-mongo-watcher/releases/latest/download/kafka-mongo-watcher-linux-386) /
    [Windows](https://github.com/etf1/kafka-mongo-watcher/releases/latest/download/kafka-mongo-watcher-windows-386.exe)
]
* Architecture **amd64** [
    [Darwin](https://github.com/etf1/kafka-mongo-watcher/releases/latest/download/kafka-mongo-watcher-darwin-amd64) /
    [Linux](https://github.com/etf1/kafka-mongo-watcher/releases/latest/download/kafka-mongo-watcher-linux-amd64) /
    [Windows](https://github.com/etf1/kafka-mongo-watcher/releases/latest/download/kafka-mongo-watcher-windows-amd64.exe)
]

### Using Docker

The watcher is also available as a [Docker image](https://hub.docker.com/r/etf1/kafka-mongo-watcher).
You can run it using the following example and pass configuration environment variables:

```bash
$ docker run \
  -e 'KAFKA_MONGO_WATCHER_REPLAY=true' \
  etf1/kafka-mongo-watcher:latest
```

### From sources

Optionally, you can also download and build it from the sources. You have to retrieve the project sources by using one of the following way:

```bash
$ go get -u github.com/etf1/kafka-mongo-watcher
# or
$ git clone https://github.com/etf1/kafka-mongo-watcher.git
```

Then, build the binary:
```bash
$ GOOS=linux GOARCH=amd64 go build -ldflags '-s -w' -o kafka-mongo-watcher ./cmd/watcher/
```

## Usage

In order to run the watcher, type the following command with the desired arguments.

You can use flags (as in this example) or environment variables:

```bash
$ ./kafka-mongo-watcher -KAFKA_MONGO_WATCHER_REPLAY=true
...
<info> Tech HTTP server started {"facility":"kafka-mongo-watcher","version":"wip","addr":":8001","file":"/usr/local/Cellar/go/1.14/libexec/src/runtime/asm_amd64.s","line":1373}
<info> Connected to mongodb database {"facility":"kafka-mongo-watcher","version":"wip","uri":"mongodb://root:toor@127.0.0.1:27011,127.0.0.1:27012,127.0.0.1:27013/watcher?replicaSet=replicaset\u0026authSource=admin"}
<info> Connected to kafka producer {"facility":"kafka-mongo-watcher","version":"wip","bootstrao-servers":"127.0.0.1:9092"}
...
```

## Available configuration variables

In dev environment you can copy `.env.dist` in `.env` and edit his content in order to customize easily the env variables.

You can set/override configuration variables from `.env` file and from `variables environment` and or from cli arguments 
(If a variables was configured in multiple sources the last will override the previous one) 

#### KAFKA_MONGO_WATCHER_CUSTOM_PIPELINE
*Type*: string

*Description*: In case you want to specify a filtering pipeline, you can specify it here. It works both wil replay and watch mode.

*Example value*: `[ { "$match": { "fullDocument.is_active": true } }, { $addFields: { "custom-field": "custom-value" } } ]`

#### KAFKA_MONGO_WATCHER_REPLAY
*Type*: bool

*Description*: In case you want to send all collection's documents once (default: false)

#### KAFKA_MONGO_WATCHER_MONGODB_URI
*Type*: string

*Description*: The MongoDB connection string URI (default: mongodb://root:toor@127.0.0.1:27011,...)

#### KAFKA_MONGO_WATCHER_MONGODB_COLLECTION_NAME
*Type*: string

*Description*: The MongoDB collection you want to watch (default: "items")

#### KAFKA_MONGO_WATCHER_MONGODB_DATABASE_NAME
*Type*: string

*Description*: The MongoDB database name you want to connect to (default: "watcher") 

#### KAFKA_MONGO_WATCHER_MONGODB_SERVER_SELECTION_TIMEOUT
*Type*: duration

*Description*: The MongoDB server selection timeout duration (default: 2s)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_BATCH_SIZE
*Type*: integer

*Description*: In case you want to enable watch batch size on MongoDB watch (default: 0 / no batch)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_FULL_DOCUMENT
*Type*: boolean

*Description*: In case you want to retrieve the full document when watching for oplogs (default: true)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_MAX_AWAIT_TIME
*Type*: duration

*Description*: In case you want to set a maximum value awaiting for new oplogs (default: 0 / don't stop)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_RESUME_AFTER
*Type*: string

*Description*: In case you want to set a logical starting point for the change stream (example : `{"_data": <hex string>}`)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_START_AT_OPERATION_TIME_I
*Type*: uint32 *(increment value)*

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_START_AT_OPERATION_TIME_T
*Type*: uint32 *(timestamp)*

*Description*: In case you want to set a timestamp for the change stream to only return changes that occurred at or after the given timestamp (default: nil)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_WATCH_MAX_RETRIES
*Type*: integer

*Description*: The max number of retries when trying to watch a collection (default: 3, set to 0 to disable retry)

#### KAFKA_MONGO_WATCHER_MONGODB_OPTION_WATCH_RETRY_DELAY
*Type*: duration

*Description*: Sleeping delay between two watch attempts (default: 500ms)

#### KAFKA_MONGO_WATCHER_KAFKA_BOOTSTRAP_SERVERS
*Type*: string

*Description*: Kafka bootstrap servers list (default: "127.0.0.1:9092")

#### KAFKA_MONGO_WATCHER_KAFKA_TOPIC
*Type*: string

*Description*: Kafka topic to write into (default: "kafka-mongo-watcher")

#### KAFKA_MONGO_WATCHER_KAFKA_PRODUCE_CHANNEL_SIZE
*Type*: integer

*Description*: The maximum size of the internal channel producer size (default: 10000)

A big value here can increase the heap memory of the application as all the payload that have to be sent to Kafka will be maintained in channel.

#### KAFKA_MONGO_WATCHER_LOG_CLI_VERBOSE
*Type*: boolean

*Description*: Used to enable/disable log verbosity (default: true)

#### KAFKA_MONGO_WATCHER_LOG_LEVEL
*Type*: string

*Description*: Used to define first level you want to start display logs (default: "info")

#### KAFKA_MONGO_WATCHER_GRAYLOG_ENDPOINT
*Type*: string

*Description*: In case you want to push logs into a Graylog server, just fill this entry with the endpoint

#### KAFKA_MONGO_WATCHER_HTTP_IDLE_TIMEOUT
*Type*: duration

*Description*: A idle timeout for HTTP technical server (default: 90s)

#### KAFKA_MONGO_WATCHER_HTTP_READ_HEADER_TIMEOUT
*Type*: duration

*Description*: A read timeout for HTTP technical server (default: 1s) 

#### KAFKA_MONGO_WATCHER_HTTP_WRITE_TIMEOUT
*Type*: duration

*Description*: A write timeout for HTTP technical server (default: 10s)

#### KAFKA_MONGO_WATCHER_HTTP_TECH_ADDR
*Type*: string

*Description*: A specified address for HTTP technical server to listen (default: ":8001")

#### KAFKA_MONGO_WATCHER_PRINT_CONFIG
*Type*: boolean

*Description*: Used to enable/disable the configuration print at startup (default: true)

#### KAFKA_MONGO_WATCHER_PPROF_ENABLED
*Type*: boolean

*Description*: In case you want to enable Go pprof debugging (default: true). No impact when not used

## Prometheus metrics

The watcher also exposes metrics about Go process and Watcher application.

These metrics can be scraped by Prometheus by browsing the following technical HTTP server endpoint: http://127.0.0.1:8001/metrics

## Run tests

Unit tests can be run with the following command:

```bash
$ go test -v -mod vendor ./...
```

And integration tests can be run with:

```bash
$ make test-integration
```

This will load needed mongodb and kafka containers and run the tests suite
