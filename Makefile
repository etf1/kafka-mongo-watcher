mocks:
	@echo "> generating mocks..."
	mockgen -source=internal/kafka/client.go -destination=internal/kafka/client_mock.go -package=kafka
	mockgen -source=internal/kafka/producer.go -destination=internal/kafka/producer_mock.go -package=kafka	
	mockgen -source=internal/mongo/client.go -destination=internal/mongo/client_mock.go -package=mongo

.PHONY: mocks
