package config

import (
	"context"
	"log"
	"os"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/heetch/confita/backend/flags"

	"github.com/etf1/go-config/dotenv"
	"github.com/etf1/go-config/env"
)

var DefaultConfigLoader = NewDefaultConfigLoader()

type Loader struct {
	backends []backend.Backend
}

func (cb *Loader) AppendBackends(backends ...backend.Backend) *Loader {
	cb.backends = append(cb.backends, backends...)

	return cb
}

func (cb *Loader) PrependBackends(backends ...backend.Backend) *Loader {
	cb.backends = append(backends, cb.backends...)

	return cb
}

func (cb *Loader) Load(ctx context.Context, to interface{}) error {
	for _, b := range cb.backends {
		if b != nil {
			err := confita.NewLoader(b).Load(ctx, to)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (cb *Loader) LoadOrFatal(ctx context.Context, to interface{}) {
	if err := cb.Load(ctx, to); err != nil {
		log.Fatal(err)
	}
}

func NewConfigLoader(backends ...backend.Backend) *Loader {
	return &Loader{backends: backends}
}

func Load(ctx context.Context, to interface{}) error {
	return DefaultConfigLoader.Load(ctx, to)
}

func LoadOrFatal(ctx context.Context, to interface{}) {
	DefaultConfigLoader.LoadOrFatal(ctx, to)
}

/*
 * Create Loader preconfigured with:
 * - .env file loader if file exist
 * - environment variable loader
 * - flags loader
 */
func NewDefaultConfigLoader() *Loader {
	builder := NewConfigLoader(
		env.NewBackend(),
		flags.NewBackend(),
	)

	f := ".env"
	if _, err := os.Stat(f); err == nil {
		builder.PrependBackends(dotenv.NewBackend(f))
	}

	return builder
}
