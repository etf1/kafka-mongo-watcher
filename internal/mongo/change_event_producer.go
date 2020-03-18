package mongo

import (
	"context"
)

type ChangeEventProducer func(ctx context.Context) (chan *ChangeEvent, error)
