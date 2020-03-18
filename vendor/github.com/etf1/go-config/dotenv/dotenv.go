package dotenv

import (
	"context"
	"fmt"
	"regexp"

	"github.com/heetch/confita/backend"
	"github.com/joho/godotenv"
)

func NewBackend(filenames ...string) backend.Backend {
	envMap, err := godotenv.Read(filenames...)
	if err != nil {
		panic(err)
	}

	return backend.Func("dotenv", func(ctx context.Context, key string) ([]byte, error) {
		matched, err := regexp.MatchString(`^\w+$`, key)
		if err != nil {
			return nil, err
		}

		if !matched {
			return nil, fmt.Errorf(`dotenv variable format expected \w+, "%s" given`, key)
		}

		if v, ok := envMap[key]; ok {
			return []byte(v), nil
		}

		return nil, backend.ErrNotFound
	})
}
