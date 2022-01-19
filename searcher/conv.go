package searcher

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"sort"
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
		Name  string `json:"name"`
		Label string `json:"label"`
		Hits  int    `json:"hits"`
	}
	// ldCtx map[string]interface{}
)

// conv converts results from the backend into corteza-discovery (jsonld-ish) format
func conv(sr *esSearchResponse, aggregation *esSearchResponse, noHits bool, moduleMeta map[string][]string, nsHandleMap map[string]string, mHandleMap map[string]string) (out *cdResults, err error) {
	if sr == nil {
		return
	}

	out = &cdResults{}
	out.Total.Value = sr.Hits.Total.Value
	out.Total.TotalOp = sr.Hits.Total.Relation
	out.Aggregations = []cdAggregation{}

	nsTotalHits := make(map[string]cdAggregationHits)
	mTotalHits := make(map[string]cdAggregationHits)

	aggsRes := sr.Aggregations
	if aggregation != nil {
		aggsRes = aggregation.Aggregations
	}
	for _, bucket := range aggsRes.Resource.Buckets {
		bucketName := getResourceName(bucket.Key)
		if bucketName == "User" {
			continue
		}

		for _, subBucket := range bucket.ResourceName.Buckets {
			resourceName := subBucket.Key
			if bucketName == "Namespace" {
				if val, is := nsTotalHits[resourceName]; is {
					val.Hits += subBucket.DocCount
					nsTotalHits[resourceName] = val
				} else {
					nsTotalHits[resourceName] = cdAggregationHits{
						Name:  resourceName,
						Label: nsHandleMap[resourceName],
						Hits:  subBucket.DocCount,
					}
				}
			}

			if bucketName == "Module" {
				if val, is := mTotalHits[resourceName]; is {
					val.Hits += subBucket.DocCount
					mTotalHits[resourceName] = val
				} else {
					mTotalHits[resourceName] = cdAggregationHits{
						Name:  resourceName,
						Label: mHandleMap[resourceName],
						Hits:  subBucket.DocCount,
					}
				}
			}
		}

		// Namespace total aggs hit counts
		for _, nsBucket := range bucket.Namespaces.Buckets {
			resourceName := nsBucket.Key
			if val, is := nsTotalHits[resourceName]; is {
				val.Hits += nsBucket.DocCount
				nsTotalHits[resourceName] = val
			} else {
				nsTotalHits[resourceName] = cdAggregationHits{
					Name:  resourceName,
					Label: nsHandleMap[resourceName],
					Hits:  nsBucket.DocCount,
				}
			}
		}

		// Module total aggs hit counts
		for _, mBucket := range bucket.Modules.Buckets {
			resourceName := mBucket.Key
			if val, is := mTotalHits[resourceName]; is {
				val.Hits += mBucket.DocCount
				mTotalHits[resourceName] = val

			} else {
				mTotalHits[resourceName] = cdAggregationHits{
					Name:  resourceName,
					Label: mHandleMap[resourceName],
					Hits:  mBucket.DocCount,
				}
			}
		}
	}

	nsAggregation := cdAggregation{
		Name:         "Namespace",
		Resource:     "compose:namespace",
		Hits:         0,
		ResourceName: []cdAggregationHits{},
	}
	for _, nsHits := range nsTotalHits {
		nsAggregation.Hits += nsHits.Hits
		nsAggregation.ResourceName = append(nsAggregation.ResourceName, nsHits)
	}
	if len(nsAggregation.ResourceName) > 0 {
		sort.Slice(nsAggregation.ResourceName, func(i, j int) bool {
			return nsAggregation.ResourceName[i].Name < nsAggregation.ResourceName[j].Name
		})
	}
	out.Aggregations = append(out.Aggregations, nsAggregation)

	mAggregation := cdAggregation{
		Name:         "Module",
		Resource:     "compose:module",
		Hits:         0,
		ResourceName: []cdAggregationHits{},
	}
	for _, mHits := range mTotalHits {
		mAggregation.Hits += mHits.Hits
		mAggregation.ResourceName = append(mAggregation.ResourceName, mHits)
	}
	if len(mAggregation.ResourceName) > 0 {
		sort.Slice(mAggregation.ResourceName, func(i, j int) bool {
			return mAggregation.ResourceName[i].Name < mAggregation.ResourceName[j].Name
		})
	}
	out.Aggregations = append(out.Aggregations, mAggregation)

	if !noHits {
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
				aux["id"] = aux["userID"]
				delete(aux, "userID")

			case "compose:record":
				// @todo: Remove below line and find proper solution for searsia as value needs to be in json
				ssVal := make(map[string]interface{})
				// fixme refactor me in the morning please
				type record struct {
					Module struct {
						Name     string `json:"name"`
						Handle   string `json:"handle"`
						ModuleId uint64 `json:"moduleId,string"`
					} `json:"module"`
					Namespace struct {
						Name        string `json:"name"`
						Handle      string `json:"handle"`
						NamespaceId uint64 `json:"namespaceId,string"`
					} `json:"namespace"`
					Values      map[string]interface{} `json:"values"`
					ValueLabels map[string]string      `json:"valueLabels"`
				}
				var r record
				if err = json.Unmarshal(h.Source, &r); err != nil {
					return
				}
				type valueJson struct {
					Name  string      `json:"name"`
					Label string      `json:"label"`
					Value interface{} `json:"value"`
				}
				key := fmt.Sprintf("%d-%d", r.Namespace.NamespaceId, r.Module.ModuleId)
				var slice []valueJson
				if val, is := moduleMeta[key]; is {
					for _, f := range val {
						slice = append(slice, valueJson{
							Name:  f,
							Label: r.ValueLabels[f],
							Value: r.Values[f],
						})

						if vv, ok := r.Values[f].([]interface{}); ok {
							if len(vv) > 0 {
								ssVal[f] = vv[0]
							}
						}
					}
				} else {
					for k, v := range r.Values {
						// @todo hardcoded value
						if len(slice) < 5 {
							slice = append(slice, valueJson{
								Name:  k,
								Label: r.ValueLabels[k],
								Value: v,
							})

							if vv, ok := v.([]interface{}); ok {
								if len(vv) > 0 {
									ssVal[k] = vv[0]
								}
							}
						}
					}
				}
				aux["customValues"] = ssVal
				aux["values"] = slice
				aux["@id"] = aux["_id"]
				delete(aux, "_id")
				delete(aux, "valueLabels")

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
	}
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
