// +build integration

package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/etf1/kafka-mongo-watcher/config"
	"github.com/etf1/kafka-mongo-watcher/internal/kafka"
	"github.com/etf1/kafka-mongo-watcher/internal/mongo"
	"github.com/etf1/kafka-mongo-watcher/internal/service"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	kafkaconfluent "gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type fixture struct {
	data     bson.M
	mongoID  string
	expected string
}

var cfg *config.Base

func setupConfigAndContainer(ctx context.Context) {
	if cfg == nil {
		os.Setenv("KAFKA_MONGO_WATCHER_PRINT_CONFIG", "false")

		cfg = config.NewBase(ctx)
		cfg.LogCliVerbose = false
		cfg.WorkerNumber = 1
	}
}

func TestMainReplay(t *testing.T) {
	// Setup application
	ctx := context.Background()
	setupConfigAndContainer(ctx)

	cfg.Replay = true
	cfg.MongoDB.CollectionName = "testreplay"
	cfg.Kafka.Topic = "integration-test-replay"

	container := service.NewContainer(cfg)

	// Given fixtures are inserted into MongoDB collection first
	fixtures := prepareFixturesDocumentsInMongoDB(ctx, t, cfg.CollectionName, container.GetMongoConnection(ctx))

	collection := container.GetMongoCollection(ctx)

	// When worker is running in replay mode
	worker := container.GetWorker()
	worker.Replay(ctx, collection, container.Cfg.Kafka.Topic)

	// Then
	assert := assert.New(t)
	assertFixturesAreInKafkaTopic(assert, cfg, fixtures)
}

func TestMainWatchAndProduce(t *testing.T) {
	// Setup application
	ctx := context.Background()
	setupConfigAndContainer(ctx)

	cfg.Replay = false
	cfg.MongoDB.CollectionName = "testwatch"
	cfg.Kafka.Topic = "integration-test-watch"

	container := service.NewContainer(cfg)

	// Given
	collection := container.GetMongoCollection(ctx)

	// When worker is running in watch mode
	worker := container.GetWorker()
	go worker.WatchAndProduce(ctx, collection, container.Cfg.Kafka.Topic)

	// And I insert fixtures in mongodb collection
	fixtures := prepareFixturesDocumentsInMongoDB(ctx, t, cfg.CollectionName, container.GetMongoConnection(ctx))

	// Then I ensure kafka topic contains the expected oplogs
	assert := assert.New(t)
	assertFixturesAreInKafkaTopic(assert, cfg, fixtures)

	// When I update fixtures in mongodb collection
	fixtures = updateFixturesDocumentsInMongoDB(ctx, t, fixtures, cfg.CollectionName, container.GetMongoConnection(ctx))

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
	consumer, err := kafkaconfluent.NewConsumer(&kafkaconfluent.ConfigMap{
		"bootstrap.servers": cfg.Kafka.BootstrapServers,
		"group.id":          cfg.Kafka.Topic,
		"auto.offset.reset": "earliest",
	})
	defer consumer.Close()

	assert.Nil(err)

	consumer.SubscribeTopics([]string{cfg.Kafka.Topic}, nil)

	for i := 0; i < len(fixtures); i++ {
		msg, err := consumer.ReadMessage(-1)
		assert.Nil(err)

		var event mongo.ChangeEvent
		bson.Unmarshal(msg.Value, event)

		var expectedEvent mongo.ChangeEvent
		bson.Unmarshal([]byte(fixtures[i].expected), expectedEvent)

		assert.Equal(expectedEvent, event)
		assert.Equal(fixtures[i].mongoID, string(msg.Key))
		assert.Equal(msg.Headers[0].Key, kafka.XTracingHeaderName)
	}
}