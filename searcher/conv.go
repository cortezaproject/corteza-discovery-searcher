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

		Hits []cdHit `json:"hits"`

		// Context ldCtx `json:"@context"`
	}

	cdHit struct {
		Type  string      `json:"@type"`
		Value interface{} `json:"@value"`
	}

	// ldCtx map[string]interface{}
)

// conv converts results from the backend into corteza-discovery (jsonld-ish) format
func conv(sr *esSearchResponse) (out *cdResults, err error) {
	out = &cdResults{}
	out.Total.Value = sr.Hits.Total.Value
	out.Total.TotalOp = sr.Hits.Total.Relation

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

		default:
			continue hits
		}

		out.Hits = append(out.Hits, cdHit{
			Type:  resType,
			Value: aux,
		})
	}

	return
}
