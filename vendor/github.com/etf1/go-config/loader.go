package config

import (
	"context"
	"log"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend"
	"github.com/heetch/confita/backend/flags"

	"github.com/etf1/go-config/dotenv"
	"github.com/etf1/go-config/env"
)

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

/*
 * Create Loader preconfigured with:
 * - .env file loader if file exist
 * - environment variable loader
 * - flags loader
 */
func NewDefaultConfigLoader() *Loader {
	return NewConfigLoader(
		dotenv.GetBackendsFromFlag()...
	).AppendBackends(
		env.NewBackend(),
		flags.NewBackend(),
	)
}
