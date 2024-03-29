//go:build integration
// +build integration

package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	kafkaconfluent "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
)

type fixture struct {
	data     bson.M
	mongoID  string
	expected string
}

var cfg *config.Base

func setupConfig(ctx context.Context) {
	if cfg == nil {
		os.Setenv("PRINT_CONFIG", "false")

		cfg = config.NewBase(ctx, configPrefix)
		cfg.LogCliVerbose = false
	}
}

func TestMainModeReplay(t *testing.T) {
	// Setup application
	ctx := context.Background()
	setupConfig(ctx)

	cfg.Replay = true
	cfg.MongoDB.CollectionName = "testreplay"
	cfg.Kafka.Topic = "integration-test-replay"

	container := service.NewContainer(cfg, ctx)
	defer container.GetKafkaRecorder().Unregister(
		container.GetMetricsRegistry(),
	)

	// Given fixtures are inserted into MongoDB collection first
	fixtures := prepareFixturesDocumentsInMongoDB(ctx, t, cfg.CollectionName, container.GetMongoConnection())

	// When worker is running in replay mode
	changeEventChan, err := container.GetChangeEventProducer()(ctx)
	if err != nil {
		panic(err)
	}
	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	container.GetKafkaClient().Produce(kafkaMessageChan)

	// Then
	assert := assert.New(t)
	assertFixturesAreInKafkaTopic(assert, cfg, fixtures)
}

func TestMainModeWatch(t *testing.T) {
	// Setup application
	ctx := context.Background()
	setupConfig(ctx)

	cfg.Replay = false
	cfg.MongoDB.CollectionName = "testwatch"
	cfg.Kafka.Topic = "integration-test-watch"

	container := service.NewContainer(cfg, ctx)
	defer container.GetKafkaRecorder().Unregister(
		container.GetMetricsRegistry(),
	)

	// When worker is running in watch mode
	changeEventChan, err := container.GetChangeEventProducer()(ctx)
	if err != nil {
		panic(err)
	}
	kafkaMessageChan := container.GetChangeEventKafkaMessageTransformer().Transform(changeEventChan)
	go container.GetKafkaClient().Produce(kafkaMessageChan)

	// And I insert fixtures in mongodb collection
	fixtures := prepareFixturesDocumentsInMongoDB(ctx, t, cfg.CollectionName, container.GetMongoConnection())

	// Then I ensure kafka topic contains the expected oplogs
	assert := assert.New(t)
	assertFixturesAreInKafkaTopic(assert, cfg, fixtures)

	// When I update fixtures in mongodb collection
	fixtures = updateFixturesDocumentsInMongoDB(ctx, t, fixtures, cfg.CollectionName, container.GetMongoConnection())

	// Then I ensure kafka topic contains the expected oplogs
	assertFixturesAreInKafkaTopic(assert, cfg, fixtures)
}

func prepareFixturesDocumentsInMongoDB(ctx context.Context, t *testing.T, collection string, connection *mongodriver.Database) []*fixture {
	fixtures := []*fixture{
		&fixture{
			data:     bson.M{"title": "my-first-item"},
			expected: `{"_id":{"_id":{"$oid":"%mongo_id%"},"copyingData":true},"operationType":"insert","fullDocument":{"_id":{"$oid":"%mongo_id%"},"title":"my-first-item"},"ns":{"db":"watcher","coll":"` + collection + `"},"documentKey":{"_id":{"$oid":"%mongo_id%"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`,
		},
		&fixture{
			data:     bson.M{"title": "my-second-amazing-item"},
			expected: `{"_id":{"_id":{"$oid":"%mongo_id%"},"copyingData":true},"operationType":"insert","fullDocument":{"_id":{"$oid":"%mongo_id%"},"title":"my-second-amazing-item"},"ns":{"db":"watcher","coll":"` + collection + `"},"documentKey":{"_id":{"$oid":"%mongo_id%"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`,
		},
		&fixture{
			data:     bson.M{"title": "my-third-item"},
			expected: `{"_id":{"_id":{"$oid":"%mongo_id%"},"copyingData":true},"operationType":"insert","fullDocument":{"_id":{"$oid":"%mongo_id%"},"title":"my-third-item"},"ns":{"db":"watcher","coll":"` + collection + `"},"documentKey":{"_id":{"$oid":"%mongo_id%"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`,
		},
	}

	for _, fixture := range fixtures {
		result, err := connection.Collection(collection).InsertOne(ctx, fixture.data)
		if err != nil {
			t.Fatal(err)
		}

		mongoID := result.InsertedID.(primitive.ObjectID).Hex()

		fixture.expected = strings.Replace(fixture.expected, "%mongo_id%", mongoID, -1)
		fixture.mongoID = mongoID
	}

	return fixtures
}

func updateFixturesDocumentsInMongoDB(ctx context.Context, t *testing.T, fixtures []*fixture, collection string, connection *mongodriver.Database) []*fixture {
	for _, fixture := range fixtures {
		objectID, err := primitive.ObjectIDFromHex(fixture.mongoID)
		if err != nil {
			t.Fatal(err)
		}

		fixture.data = bson.M{"title": "my-new-updated-title"}
		fixture.expected = `{"_id":{"_id":{"$oid":"%mongo_id%"},"copyingData":true},"operationType":"update","fullDocument":{"_id":{"$oid":"%mongo_id%"},"title":"my-new-updated-title"},"ns":{"db":"watcher","coll":"` + collection + `"},"documentKey":{"_id":{"$oid":"%mongo_id%"}},"clusterTime":{"$date":{"$numberLong":"-62135596800000"}}}`

		_, err = connection.Collection(collection).UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": fixture.data})
		if err != nil {
			t.Fatal(err)
		}
	}

	return fixtures
}

func assertFixturesAreInKafkaTopic(assert *assert.Assertions, cfg *config.Base, fixtures []*fixture) {
	defer time.Sleep(1 * time.Second) // Ensure consumer is closed

	consumer, err := kafkaconfluent.NewConsumer(&kafkaconfluent.ConfigMap{
		"bootstrap.servers": cfg.Kafka.BootstrapServers,
		"group.id":          cfg.Kafka.Topic,
		"auto.offset.reset": "earliest",
	})

	assert.Nil(err)

	consumer.SubscribeTopics([]string{cfg.Kafka.Topic}, nil)

	for i := 0; i < len(fixtures); i++ {
		var msg *kafkaconfluent.Message
		var err error

		for {
			msg, err = consumer.ReadMessage(30 * time.Second)
			if err == nil {
				break
			}

			// In case this is not a "unknown topic" error, break and fail, elsewhere we will wait for the topic to be ready
			// and retry
			if kafkaError, ok := err.(kafkaconfluent.Error); ok && kafkaError.Code() != kafkaconfluent.ErrUnknownTopicOrPart {
				break
			}
		}

		assert.Nil(err)

		var event mongo.ChangeEvent
		bson.Unmarshal(msg.Value, event)

		var expectedEvent mongo.ChangeEvent
		bson.Unmarshal([]byte(fixtures[i].expected), expectedEvent)

		assert.Equal(expectedEvent, event)
		assert.Equal(fixtures[i].mongoID, string(msg.Key))
		assert.Equal(msg.Headers[0].Key, kafka.XTracingHeaderName)
	}

	consumer.Close()
}
