package searcher

import (
	"encoding/json"
	"github.com/spf13/cast"
)

type (
	// corteza discovery results
	cdResults struct {
		Total struct {
			Value   int    `json:"value"`
			TotalOp string `json:"op"`
		} `json:"total"`

		Hits         []cdHit         `json:"hits"`
		TotalHits    int             `json:"total_hits"`
		Aggregations []cdAggregation `json:"aggregations"`

		// Context ldCtx `json:"@context"`
	}

	cdHit struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}

	cdAggregation struct {
		Resource     string              `json:"resource"`
		Name         string              `json:"name"`
		Hits         int                 `json:"hits"`
		ResourceName []cdAggregationHits `json:"resource_name"`
	}

	cdAggregationHits struct {
		Name string `json:"name"`
		Hits int    `json:"hits"`
	}
	// ldCtx map[string]interface{}
)

// conv converts results from the backend into corteza-discovery (jsonld-ish) format
func conv(sr *esSearchResponse) (out *cdResults, err error) {
	if sr == nil {
		return
	}

	out = &cdResults{}
	out.Total.Value = sr.Hits.Total.Value
	out.Total.TotalOp = sr.Hits.Total.Relation
	out.Aggregations = []cdAggregation{}

	for _, bucket := range sr.Aggregations.Resource.Buckets {
		if bucket.Key == "compose:record" || bucket.Key == "system:user" {
			continue
		}
		var resourceNames []cdAggregationHits
		for _, subBucket := range bucket.ResourceName.Buckets {
			resourceNames = append(resourceNames, cdAggregationHits{
				Name: subBucket.Key,
				Hits: subBucket.DocCount,
			})
		}

		out.Aggregations = append(out.Aggregations, cdAggregation{
			Name:         getResourceName(bucket.Key),
			Hits:         bucket.DocCount,
			ResourceName: resourceNames,
		})
	}

hits:
	for _, h := range sr.Hits.Hits {
		aux := map[string]interface{}{}
		if err = json.Unmarshal(h.Source, &aux); err != nil {
			return
		}

		resType := cast.ToString(aux["resourceType"])
		delete(aux, "resourceType")
		switch resType {
		case "system:user":
			aux["@id"] = aux["userID"]
			delete(aux, "userID")

		case "compose:record":
			aux["@id"] = aux["_id"]
			delete(aux, "_id")

		case "compose:namespace":
			aux["@id"] = aux["_id"]
			delete(aux, "_id")

		case "compose:module":
			aux["@id"] = aux["_id"]
			delete(aux, "_id")

		default:
			continue hits
		}

		out.Hits = append(out.Hits, cdHit{
			Type:  resType,
			Value: aux,
		})
	}

	out.TotalHits = len(out.Hits)
	return
}

// @todo use RBAC resource stringify
// getResourceName return name of resource based on resource type
func getResourceName(resType string) string {
	switch resType {
	case "system:user":
		return "User"
	case "compose:record":
		return "Record"
	case "compose:namespace":
		return "Namespace"
	case "compose:module":
		return "Module"
	default:
		return "Resource"
	}
}
