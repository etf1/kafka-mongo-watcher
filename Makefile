mocks:
	@echo "> generating mocks..."
	mockgen -source=internal/kafka/client.go -destination=internal/kafka/client_mock.go -package=kafka
	mockgen -source=internal/kafka/producer.go -destination=internal/kafka/producer_mock.go -package=kafka
	mockgen -source=internal/metrics/kafka.go -destination=internal/metrics/kafka_mock.go -package=metrics
	mockgen -source=internal/mongo/client.go -destination=internal/mongo/client_mock.go -package=mongo
	mockgen -source=internal/mongo/collection.go -destination=internal/mongo/collection_mock.go -package=mongo

clean:
	@echo "> cleaning..."
	docker-compose down

init:
	@echo "> initialization..."
	docker-compose up -d

	@echo "> waiting for mongodb replication to be ready (this can take some seconds...)"
	-for i in {1..5}; do docker-compose exec mongo-primary mongo \
		'mongodb://root:toor@127.0.0.1:27011,127.0.0.1:27012,127.0.0.1:27013/watcher?connect=replicaset&replicaSet=replicaset&authSource=admin' \
		--eval 'rs.status();' > /dev/null && break || sleep 1; done

run-test-integration:
	@echo "> running integration tests..."
	-go test -v -count=1 -mod vendor -tags integration ./cmd/watcher

test-integration: init run-test-integration clean

.PHONY: clean init mocks test-integration run-test-integration
