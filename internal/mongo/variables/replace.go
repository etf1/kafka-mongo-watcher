package variables

import (
	"strconv"
	"strings"
	"time"
)

type valueGetterFunc func() string

var (
	// Contains the time.Now function that returns current time.
	// It is declared as a variable for tests to be able to declare a special current time.
	now = time.Now

	// Contains the available variables to be replaced when using a custom aggregation pipeline.
	replacements = map[string]valueGetterFunc{
		"%currentTimestamp%": func() string {
			return strconv.FormatInt(now().Unix()*1000, 10)
		},
	}
)

func Replace(json string) string {
	for variable, valueGetterFunc := range replacements {
		json = strings.ReplaceAll(json, variable, valueGetterFunc())
	}

	return json
}
