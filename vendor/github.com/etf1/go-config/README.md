# e-TF1 go-config

Package e-TF1 go-config allows you to load your application configuration from multiple sources (dotenv, env, flags...).
This package is built on top of [confita](https://github.com/heetch/confita)

The main differences with confita are the following:

 - all the backend matched value is written on a struct (confita will only load the last backend that matched at least one value)
 - the last backend that matches the config key will override the previous match
 - we allow to override by empty string

# Quick Usage

Create your own struct with your configuration values

```go
package main

import (
	"context"
	"fmt"

	"github.com/etf1/go-config"
)

type MyConfig struct {
	Debug    bool   `config:"DEBUG"`
	HTTPAddr string `config:"HTTP_ADDR"`
	BDDPassword string `config:"BDD_PASSWORD" print:"-"` // with print:"-" it will be print as "*** Hidden value ***"
}

func New(context context.Context) *MyConfig {
	cfg := &MyConfig{
		HTTPAddr: ":8001",
		BDDPassword: "my_fake_password",
	}

	config.NewDefaultConfigLoader().LoadOrFatal(context, cfg)
	return cfg
}

func main() {
	cfg := New(context.Background())
	
	fmt.Println(config.TableString(cfg))
}
```

It will print something similar to

```
-----------------------------------
       Debug|                false|bool `config:"DEBUG"`
    HTTPAddr|                :8001|string `config:"HTTP_ADDR"`
 BDDPassword| *** Hidden value ***|string `config:"BDD_PASSWORD" print:"-"`
-----------------------------------
```

# Loaders

The library provides a DefaultConfigLoader that loads from

- .env (if file ./.env was found)
- environment variables
- flags 

You can create your own loader chain

```go
// create your own chain loader
cl := config.NewConfigLoader(
    file.NewBackend("myfile.yml"),
    flags.NewBackend(),
)
```

```go
// you can even append multiple backends to it
cl.AppendBackends(
    file.NewBackend("myfile.json"),
)
```

```go
// and even prepend multiple backends to it
f := ".env"
if _, err := os.Stat(f); err == nil {
    cl.PrependBackends(dotenv.NewBackend(f))
}
```

# Custom backends

We use all built-in Confita backends and also:

* `prefix`: A backend that allows to prefix configuration keys with a special value (useful when using environment variables on generic projects)

# Printer

The library provides you a config table printer. A special tag `print:"-"` prevents config value to be printed.

```
-----------------------------------
       Debug|                false|bool   `config:"DEBUG"`
    HTTPAddr|                :8001|string `config:"HTTP_ADDR"`
 BDDPassword| *** Hidden value ***|string `config:"BDD_PASSWORD" print:"-"`
-----------------------------------
```
