package variables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReplace(t *testing.T) {
	now = func() time.Time {
		return time.Date(2021, time.June, 8, 18, 0, 0, 0, time.UTC)
	}

	testCases := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "Empty string",
			json:     "",
			expected: "",
		},
		{
			name:     "Single variable",
			json:     `[ { "$match": { "date": { "$gt": "%currentTimestamp%" } } } ]`,
			expected: `[ { "$match": { "date": { "$gt": "1623175200000" } } } ]`,
		},
		{
			name:     "Multiple variable",
			json:     `[ { "$match": { "date": { "$gt": "%currentTimestamp%" } } }, { "$match": { "end": { "$lt": "%currentTimestamp%" } } } ]`,
			expected: `[ { "$match": { "date": { "$gt": "1623175200000" } } }, { "$match": { "end": { "$lt": "1623175200000" } } } ]`,
		},
		{
			name:     "No variable",
			json:     `[ { "$match": { "date": { "$gt": "1623175200000" } } }, { "$match": { "end": { "$lt": "1623175200000" } } } ]`,
			expected: `[ { "$match": { "date": { "$gt": "1623175200000" } } }, { "$match": { "end": { "$lt": "1623175200000" } } } ]`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, Replace(testCase.json))
		})
	}
}
