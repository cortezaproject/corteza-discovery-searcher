package searcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/go-chi/jwtauth"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type (
	esSearchParamsIndex struct {
		Prefix struct {
			Index struct {
				Value string `json:"value"`
			} `json:"_index"`
		} `json:"prefix"`
	}

	esSimpleQueryString struct {
		Wrap struct {
			Query string `json:"query"`
		} `json:"simple_query_string"`
	}

	esSearchParams struct {
		Query struct {
			Bool struct {
				// query context
				Must []interface{} `json:"must"`

				// filter context
				Filter  []interface{} `json:"filter,omitempty"`
				MustNot []interface{} `json:"must_not,omitempty"`
			} `json:"bool"`
		} `json:"query"`

		Aggregations EsSearchAggrTerms `json:"aggs,omitempty"`
	}

	esSearchAggrTerm struct {
		Field string `json:"field"`
		Size  int    `json:"size,omitempty"`
	}

	esSearchAggrComposite struct {
		Sources interface{} `json:"sources"` // it can be esSearchAggrTerm,.. (Histogram, Date histogram, GeoTile grid)
		Size    int         `json:"size,omitempty"`
	}

	esSearchAggr struct {
		Terms        esSearchAggrTerm  `json:"terms"`
		Aggregations EsSearchAggrTerms `json:"aggs,omitempty"`
		//Composite *esSearchAggrComposite `json:"composite"`
	}

	esSearchResponse struct {
		Took         int                  `json:"took"`
		TimedOut     bool                 `json:"timed_out"`
		Hits         esSearchHits         `json:"hits"`
		Aggregations esSearchAggregations `json:"aggregations"`
	}

	esSearchTotal struct {
		Value    int    `json:"value"`
		Relation string `json:"relation"`
	}

	esSearchHits struct {
		Total esSearchTotal  `json:"total"`
		Hits  []*esSearchHit `json:"hits"`
	}

	esSearchHit struct {
		Index  string          `json:"_index"`
		ID     string          `json:"_id"`
		Source json.RawMessage `json:"_source"`
	}

	esSearchAggregations struct {
		Resource struct {
			DocCountErrorUpperBound int `json:"-"`
			SumOtherDocCount        int `json:"-"`
			Buckets                 []struct {
				Key          string `json:"key"`
				DocCount     int    `json:"doc_count"`
				ResourceName struct {
					DocCountErrorUpperBound int `json:"-"`
					SumOtherDocCount        int `json:"-"`
					Buckets                 []struct {
						Key      string `json:"key"`
						DocCount int    `json:"doc_count"`
					} `json:"buckets"`
				} `json:"resourceName"`
				Namespaces struct {
					DocCountErrorUpperBound int `json:"-"`
					SumOtherDocCount        int `json:"-"`
					Buckets                 []struct {
						Key      string `json:"key"`
						DocCount int    `json:"doc_count"`
					} `json:"buckets"`
				} `json:"namespaces"`
				Modules struct {
					DocCountErrorUpperBound int `json:"-"`
					SumOtherDocCount        int `json:"-"`
					Buckets                 []struct {
						Key      string `json:"key"`
						DocCount int    `json:"doc_count"`
					} `json:"buckets"`
				} `json:"modules"`
			} `json:"buckets"`
		} `json:"resource"`
	}

	searchParams struct {
		query         string
		moduleAggs    []string
		namespaceAggs []string
		dumpRaw       bool
		size          int
	}
)

func EsClient(aa []string) (*elasticsearch.Client, error) {
	return elasticsearch.NewClient(elasticsearch.Config{
		Addresses:            aa,
		EnableRetryOnTimeout: true,
		MaxRetries:           5,
	})
}

func search(ctx context.Context, esc *elasticsearch.Client, log *zap.Logger, p searchParams) (*esSearchResponse, error) {
	var (
		buf          bytes.Buffer
		roles        []string
		userID       uint64
		_, claims, _ = jwtauth.FromContext(ctx)
	)

	if _, has := claims["roles"]; has {
		if rolesStr, is := claims["roles"].(string); is {
			roles = strings.Split(rolesStr, " ")
		}
	}
	if _, has := claims["sub"]; has {
		if sub, is := claims["sub"].(string); is {
			userID, _ = strconv.ParseUint(sub, 10, 64)
		}
	}

	sqs := esSimpleQueryString{}
	sqs.Wrap.Query = p.query

	query := esSearchParams{}
	index := esSearchParamsIndex{}

	// Decide what indexes we can use
	if userID == 0 {
		// Missing, invalid, expired access token (JWT)
		//index.Prefix.Index.Value = "corteza-public-"
		// fixme revert this, temp fix for searching
		index.Prefix.Index.Value = "corteza-private-"
	} else {
		// Authenticated user
		index.Prefix.Index.Value = "corteza-private-"

		query.Query.Bool.Filter = []interface{}{
			//map[string]map[string]interface{}{
			//	"exists": {"field": []string{"security.allowedRoles", "security.deniedRoles"}},
			//},
			map[string]map[string]interface{}{
				// Skip all documents that do not have baring roles in the allow list
				"terms": {"security.allowedRoles": roles},
			},
		}
		query.Query.Bool.MustNot = []interface{}{
			map[string]map[string]interface{}{
				// Skip all documents that have baring roles in the deny list
				"terms": {"security.deniedRoles": roles},
			},
		}
		_ = roles
	}

	query.Query.Bool.Must = []interface{}{index}
	if len(p.query) > 0 {
		query.Query.Bool.Must = append(query.Query.Bool.Must, sqs)
	}

	// Aggregations V1.0
	//if len(p.aggregations) > 0 {
	//	query.Aggregations = make(map[string]esSearchAggr)
	//
	//	for _, a := range p.aggregations {
	//		query.Aggregations[a] = esSearchAggr{esSearchAggrTerm{Field: a + ".keyword"}}
	//	}
	//}

	// Aggregations V1.0 Improved fixme
	//if len(p.aggregations) > 0 {
	//	for _, a := range p.aggregations {
	//		if len(a) > 0 {
	//			sqs = esSimpleQueryString{}
	//			sqs.Wrap.Query = a
	//			query.Query.Bool.Must = append(query.Query.Bool.Must, sqs)
	//		}
	//	}
	//}

	// Here is how aggs should look like for elastic search
	//"aggs": {
	//	"resource": {
	//		"terms": {
	//			"field": "resourceType.keyword"
	//		},
	//		"aggs": {
	//			"resourceName": {
	//				"terms": {
	//					"field": "name.keyword"
	//				}
	//			}
	//		}
	//	}
	//}
	query.Aggregations = make(map[string]esSearchAggr)
	query.Aggregations["resource"] = esSearchAggr{
		Terms: esSearchAggrTerm{
			Field: "resourceType.keyword",
			Size:  999,
		},
		Aggregations: EsSearchAggrTerms{
			"resourceName": esSearchAggr{
				Terms: esSearchAggrTerm{
					Field: "name.keyword",
					Size:  999,
				},
			},
			"modules": esSearchAggr{
				Terms: esSearchAggrTerm{
					Field: "module.name.keyword",
					Size:  999,
				},
			},
			"namespaces": esSearchAggr{
				Terms: esSearchAggrTerm{
					Field: "namespace.name.keyword",
					Size:  999,
				},
			},
		},
	}

	// Aggregations V2.0
	//if len(p.aggregations) > 0 {
	//	query.Aggregations = (Aggregations{}).encodeTerms(p.aggregations)
	//}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("could not encode query: %q", err)
	}

	// Why set size to 999? default value for size is 10,
	// so we needed to set value till we add (@todo) pagination to search result
	if p.size == 0 {
		p.size = 999
	}

	sReqArgs := []func(*esapi.SearchRequest){
		esc.Search.WithContext(ctx),
		esc.Search.WithBody(&buf),
		esc.Search.WithTrackTotalHits(true),
		//esc.Search.WithScroll(),
		esc.Search.WithSize(p.size),
		//esc.Search.WithFrom(0), // paging (offset)
		//esc.Search.WithExplain(true), // debug
	}

	if p.dumpRaw {
		sReqArgs = append(
			sReqArgs,
			esc.Search.WithSourceExcludes("security"),
			esc.Search.WithPretty(),
		)
	}

	// Perform the search request.
	res, err := esc.Search(sReqArgs...)

	if err != nil {
		return nil, err
	}

	if err = validElasticResponse(log, res, err); err != nil {
		return nil, fmt.Errorf("invalid search response: %w", err)
	}

	defer res.Body.Close()

	if p.dumpRaw {
		// Copy body buf and then restore it
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		os.Stdout.Write(bodyBytes)
	}

	var sr = &esSearchResponse{}
	if err = json.NewDecoder(res.Body).Decode(sr); err != nil {
		return nil, err
	}

	// Print the response status, number of results, and request duration.
	log.Debug("search completed",
		zap.String("query", sqs.Wrap.Query),
		zap.String("indexPrefix", index.Prefix.Index.Value),
		zap.String("status", res.Status()),
		zap.Int("took", sr.Took),
		zap.Bool("timedOut", sr.TimedOut),
		zap.Int("hits", sr.Hits.Total.Value),
		zap.String("hitsRelation", sr.Hits.Total.Relation),
	)

	return sr, nil
}

func validElasticResponse(log *zap.Logger, res *esapi.Response, err error) error {
	if err != nil {
		return fmt.Errorf("failed to get response from search backend: %w", err)
	}

	if res.IsError() {
		defer res.Body.Close()
		var rsp struct {
			Error struct {
				Type   string
				Reason string
			}
		}

		if err := json.NewDecoder(res.Body).Decode(&rsp); err != nil {
			return fmt.Errorf("could not parse response body: %w", err)
		} else {
			return fmt.Errorf("search backend responded with an error: %s (type: %s, status: %s)", rsp.Error.Reason, rsp.Error.Type, res.Status())
		}
	}

	return nil
}
