package prefix

import (
	"context"

	"github.com/heetch/confita/backend"
)

// Option is used to configure the prefix backend.
type Option func(*Backend)

// Backend loads keys using prefix.
type Backend struct {
	delimiter string
	prefix    string
	backend   backend.Backend
}

// NewBackend creates a new prefix backend.
func NewBackend(prefix string, backend backend.Backend, opts ...Option) *Backend {
	b := &Backend{
		delimiter: "_",
		prefix:    prefix,
		backend:   backend,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// WithDelimiter allows to specify a delimiter between prefix and key.
func WithDelimiter(delimiter string) Option {
	return func(b *Backend) {
		b.delimiter = delimiter
	}
}

// Get loads the given key using specified key prefix and backend.
func (b *Backend) Get(ctx context.Context, key string) ([]byte, error) {
	return b.backend.Get(ctx, b.prefix+b.delimiter+key)
}

// Name returns the name of the backend.
func (b *Backend) Name() string {
	return "prefix_" + b.backend.Name()
}
