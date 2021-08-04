package searcher

import (
	"errors"
	"fmt"
)

type (
	Aggregations struct{}

	// EsSearchAggrTerms is aggregations parameter for es search api.
	EsSearchAggrTerms map[string]esSearchAggr

	// EsSearchAggrResults is a list of EsSearchAggrResult.
	EsSearchAggrResults map[string]EsSearchAggrResult

	// EsSearchAggrResult contains the result of the TermAggregation.
	EsSearchAggrResult struct {
		Buckets []Bucket `json:"buckets"`
	}

	// Bucket contains how often a specific key was found in a term aggregation.
	Bucket struct {
		Key   interface{} `json:"key"`
		Count int         `json:"doc_count"`
	}
)

// @todo: Import/export for esSearch structs
// encodeTerms takes a list of aggregations search string and returns EsSearchAggrTerms
func (t Aggregations) encodeTerms(aggregations []string) (res EsSearchAggrTerms) {
	res = make(map[string]esSearchAggr)
	for _, a := range aggregations {
		res[a] = esSearchAggr{esSearchAggrTerm{Field: a + ".keyword"}}
	}

	return
}

// RangeAggregate returns the min- and max-value for a specific field in a specific index
func RangeAggregate(index, doctype string, query map[string]interface{}, field string) (float64, float64, error) {
	request := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"min_" + field: map[string]interface{}{
				"min": map[string]interface{}{
					"field": field,
				},
			},
			"max_" + field: map[string]interface{}{
				"max": map[string]interface{}{
					"field": field,
				},
			},
		},
	}
	if query != nil {
		request["query"] = query
	}

	// call search

	result := struct {
		Aggregations map[string]struct {
			Value float64 `json:"value"`
		} `json:"aggregations"`
	}{}

	if result.Aggregations == nil {
		return 0, 0, fmt.Errorf("no aggregation result found")
	}
	minValue, ok1 := result.Aggregations["min_"+field]
	maxValue, ok2 := result.Aggregations["max_"+field]
	if !ok1 || !ok2 {
		return 0, 0, errors.New("min or max value not a number")
	}
	return minValue.Value, maxValue.Value, nil
}

// CardinalityAggregate returns the unique count of a specific field in a specific index
func CardinalityAggregate(index, doctype string, query map[string]interface{}, field string) (int64, error) {
	request := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"count_" + field: map[string]interface{}{
				"cardinality": map[string]interface{}{
					"field": field,
				},
			},
		},
	}
	if query != nil {
		request["query"] = query
	}

	// call search

	result := struct {
		Aggregations map[string]struct {
			Value int64 `json:"value"`
		} `json:"aggregations"`
	}{}
	value, ok := result.Aggregations["count_"+field]
	if !ok {
		return 0, errors.New("could not find count of field")
	}
	return value.Value, nil
}

var compositeSize = 500

// @todo to utility
func compositeAggregateAfter(query map[string]interface{}, field string, after interface{}) ([]*Bucket, error) {
	var compositeResult []*Bucket
	request := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"my_buckets": map[string]interface{}{
				"composite": map[string]interface{}{
					"size": compositeSize,
					"sources": map[string]interface{}{
						field: map[string]interface{}{
							"terms": map[string]interface{}{
								"field": field,
							},
						},
					},
				},
			},
		},
	}
	if after != nil {
		request["aggs"].(map[string]interface{})["my_buckets"].(map[string]interface{})["composite"].(map[string]interface{})["after"] = after
	}
	if query != nil {
		request["query"] = query
	}

	// call search

	_ = struct {
		Aggregations struct {
			MyBuckets struct {
				Buckets []*struct {
					Key   map[string]interface{} `json:"key"`
					Count int                    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"my_buckets"`
		} `json:"aggregations"`
	}{}

	return compositeResult, nil
}

type DateHistogramInterval string

const (
	DateHistogramIntervalYear   = "year"
	DateHistogramIntervalMonth  = "month"
	DateHistogramIntervalDay    = "day"
	DateHistogramIntervalHour   = "hour"
	DateHistogramIntervalMinute = "minute"
	DateHistogramIntervalSecond = "second"
	DateHistogramIntervalAuto   = "auto"
)

// DateHistogramAggregate fixme to utility
func DateHistogramAggregate(query map[string]interface{}, field string, interval DateHistogramInterval, buckets int) ([]*Bucket, error) {
	var dateHistogramResult []*Bucket
	var request map[string]interface{}
	if interval == DateHistogramIntervalAuto {
		request = map[string]interface{}{
			"size": 0,
			"aggs": map[string]interface{}{
				"my_datehistogram": map[string]interface{}{
					"auto_date_histogram": map[string]interface{}{
						"field":   field,
						"buckets": buckets,
					},
				},
			},
		}
	} else {
		request = map[string]interface{}{
			"size": 0,
			"aggs": map[string]interface{}{
				"my_datehistogram": map[string]interface{}{
					"date_histogram": map[string]interface{}{
						"field":    field,
						"interval": string(interval),
					},
				},
			},
		}
	}
	if query != nil {
		request["query"] = query
	}

	// call search
	result := struct {
		Aggregations struct {
			MyDateHistogram struct {
				DateHistogram []*struct {
					Key   int64 `json:"key"`
					Count int   `json:"doc_count"`
				} `json:"buckets"`
			} `json:"my_datehistogram"`
		} `json:"aggregations"`
	}{}

	for _, bucket := range result.Aggregations.MyDateHistogram.DateHistogram {
		dateHistogramResult = append(dateHistogramResult, &Bucket{Key: bucket.Key, Count: bucket.Count})
	}
	return dateHistogramResult, nil
}
