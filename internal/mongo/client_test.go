package mongo

import (
	"testing"

	"github.com/gol4ng/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	clientTest := newClient()

	assert.IsType(t, new(logger.Logger), clientTest.logger)
}

func TestNewClientWithLogger(t *testing.T) {
	logger := logger.NewNopLogger()
	clientTest := newClient(WithLogger(logger))

	assert.Equal(t, logger, clientTest.logger)
}
