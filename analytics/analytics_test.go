package analytics

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/stretchr/testify/assert"
)

const (
	requestTemplate  = "POST / HTTP/1.1\r\nHost: localhost:8281\r\nUser-Agent: test-agent\r\nContent-Length: %d\r\n\r\n%s"
	responseTemplate = "HTTP/0.0 200 OK\r\nContent-Length: %d\r\nConnection: close\r\nContent-Type: application/json\r\n\r\n%s"
)

const sampleSchema = `
type Query {
  characters(filter: FilterCharacter, page: Int): Characters
  listCharacters(): [Characters]!
}
input FilterCharacter {
  name: String
  status: String
  species: String
  type: String
  gender: String! = "M"
}
type Characters {
  info: Info
  results: [Character]
}
type Info {
  count: Int
  next: Int
  pages: Int
  prev: Int
}
type Character {
  gender: String
  id: ID
  name: String
}`

func TestAnalyticsRecord_IsGraphRecord(t *testing.T) {
	t.Run("should return false when no tags are available", func(t *testing.T) {
		record := AnalyticsRecord{}
		assert.False(t, record.IsGraphRecord())
	})

	t.Run("should return false when tags do not contain the graph analytics tag", func(t *testing.T) {
		record := AnalyticsRecord{
			Tags: []string{"tag_1", "tag_2", "tag_3"},
		}
		assert.False(t, record.IsGraphRecord())
	})

	t.Run("should return true when tags contain the graph analytics tag", func(t *testing.T) {
		record := AnalyticsRecord{
			Tags: []string{"tag_1", "tag_2", PredefinedTagGraphAnalytics, "tag_4", "tag_5"},
		}
		assert.True(t, record.IsGraphRecord())
	})
}

func TestAnalyticsRecord_ToGraphRecord(t *testing.T) {
	recordSample := AnalyticsRecord{
		Method:       "POST",
		Host:         "localhost:8281",
		ResponseCode: 200,
		Path:         "/",
		RawPath:      "/",
		Day:          1,
		Month:        1,
		Year:         2022,
		Hour:         0,
		TimeStamp:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		APIName:      "test-api",
		APIID:        "test-api",
		ApiSchema:    base64.StdEncoding.EncodeToString([]byte(sampleSchema)),
	}
	graphRecordSample := GraphRecord{
		AnalyticsRecord: recordSample,
		Types:           make(map[string][]string),
	}

	testCases := []struct {
		title    string
		request  string
		response string
		expected func(string, string) GraphRecord
	}{
		{
			title:    "no error",
			request:  `{"query":"query{\n  characters(filter: {\n    \n  }){\n    info{\n      count\n    }\n  }\n}"}`,
			response: `{"data":{"characters":{"info":{"count":758}}}}`,
			expected: func(request, response string) GraphRecord {
				g := graphRecordSample
				g.HasErrors = false
				g.Types = map[string][]string{
					"Characters": {"info"},
					"Info":       {"count"},
				}
				g.OperationType = "query"
				return g
			},
		},
		{
			title:    "no error list operation",
			request:  `{"query":"query{\n  listCharacters(){\n    info{\n      count\n    }\n  }\n}"}`,
			response: `{"data":{"characters":{"info":{"count":758}}}}`,
			expected: func(request, response string) GraphRecord {
				g := graphRecordSample
				g.HasErrors = false
				g.Types = map[string][]string{
					"Characters": {"info"},
					"Info":       {"count"},
				}
				g.OperationType = "query"
				return g
			},
		},
		{
			title:    "has variables",
			request:  `{"query":"query{\n  characters(filter: {\n    \n  }){\n    info{\n      count\n    }\n  }\n}","variables":{"a":"test"}}`,
			response: `{"data":{"characters":{"info":{"count":758}}}}`,
			expected: func(request, response string) GraphRecord {
				g := graphRecordSample
				g.HasErrors = false
				g.Types = map[string][]string{
					"Characters": {"info"},
					"Info":       {"count"},
				}
				g.OperationType = "query"
				g.Variables = base64.StdEncoding.EncodeToString([]byte(`{"a":"test"}`))
				return g
			},
		},
		{
			title:   "has errors",
			request: `{"query":"query{\n  characters(filter: {\n    \n  }){\n    info{\n      count\n    }\n  }\n}"}`,
			response: `{
  "errors": [
    {
      "message": "Name for character with ID 1002 could not be fetched.",
      "locations": [{ "line": 6, "column": 7 }],
      "path": ["hero", "heroFriends", 1, "name"]
    }
  ]
}`,
			expected: func(request, response string) GraphRecord {
				g := graphRecordSample
				g.HasErrors = true
				g.Types = map[string][]string{
					"Characters": {"info"},
					"Info":       {"count"},
				}
				g.OperationType = "query"
				g.Errors = append(g.Errors, graphError{
					Message: "Name for character with ID 1002 could not be fetched.",
					Path:    []interface{}{"hero", "heroFriends", float64(1), "name"},
				})
				return g
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.title, func(t *testing.T) {
			a := recordSample
			a.RawRequest = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(
				requestTemplate,
				len(testCase.request),
				testCase.request,
			)))
			a.RawResponse = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(
				responseTemplate,
				len(testCase.response),
				testCase.response,
			)))
			expected := testCase.expected(testCase.request, testCase.response)
			expected.AnalyticsRecord = a
			gotten, err := a.ToGraphRecord()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, gotten, cmpopts.IgnoreFields(GraphRecord{}, "RawRequest", "RawResponse")); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
