package dotenv

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/heetch/confita/backend"
	"github.com/joho/godotenv"
)

var (
	envFiles string
	o        sync.Once
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

func GetBackendsFromFlag() []backend.Backend {
	o.Do(func() {
		flag.StringVar(&envFiles, "config-env-files", ".env", "dot env file path separate by \",\" last file will override previous one\neg: application -config-env-files=\".env,.env.test\"\nwill load .env file then override with .env.test\n")
		flag.Parse()
	})
	return GetBackends(strings.Split(envFiles, ",")...)
}

func GetBackends(paths ...string) []backend.Backend {
	var backends []backend.Backend
	for _, dotFile := range paths {
		if f, err := os.Stat(dotFile); err == nil && !f.IsDir() {
			backends = append(backends, NewBackend(dotFile))
		}
	}
	return backends
}
