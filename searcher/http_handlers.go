package searcher

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var _ = spew.Dump

type (
	handlers struct {
		log *zap.Logger
		esc *elasticsearch.Client
		api *apiClient
	}
)

func Handlers(r chi.Router, log *zap.Logger, esc *elasticsearch.Client, api *apiClient) *handlers {
	h := &handlers{
		esc: esc,
		log: log,
		api: api,
	}

	r.Use()

	r.Get("/healthcheck", h.Healthcheck)
	r.Get("/sandbox", h.Sandbox)
	r.Get("/", h.Search)
	//r.Get("/suggest", h.Suggest)

	return h
}

func (h handlers) Healthcheck(w http.ResponseWriter, r *http.Request) {
	res, err := h.esc.Ping(
		h.esc.Ping.WithContext(r.Context()),
	)

	if validElasticResponse(h.log, res, err) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "unhealthy")
		return
	}

	defer res.Body.Close()

	_, _ = fmt.Fprintf(w, "healthy")
}

func (h handlers) Sandbox(w http.ResponseWriter, r *http.Request) {
	p := "." + r.URL.Path
	if p == "./" {
		p = "./sandbox/index.html"
	}
	http.ServeFile(w, r, p)
}

func (h handlers) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = r.ParseForm()

	size := 0
	if len(r.FormValue("size")) > 0 {
		size, _ = strconv.Atoi(r.FormValue("size"))
	}

	//searchString := r.FormValue("q")
	var (
		// fixme cleanup make struct or something
		searchString  = r.FormValue("q")
		moduleAggs    = r.Form["moduleAggs"]
		namespaceAggs = r.Form["namespaceAggs"]

		results     *esSearchResponse
		aggregation *esSearchResponse
		err         error
	)
	results, err = search(r.Context(), h.esc, h.log, searchParams{
		query:         searchString,
		moduleAggs:    moduleAggs,
		namespaceAggs: namespaceAggs,
		size:          size,
		dumpRaw:       r.FormValue("dump") != "",
	})

	if err != nil {
		h.log.Error("could not execute search", zap.Error(err))
	}

	if len(searchString) == 0 {
		aggregation, err = search(r.Context(), h.esc, h.log, searchParams{
			size:    size,
			dumpRaw: r.FormValue("dump") != "",
		})
		if err != nil {
			h.log.Error("could not execute aggregation search", zap.Error(err))
		}
	}

	noHits := len(searchString) == 0 && len(searchString) == 0 && len(namespaceAggs) == 0
	if cres, err := conv(results, aggregation, noHits); err != nil {
		h.log.Error("could not encode response body", zap.Error(err))
	} else if err = json.NewEncoder(w).Encode(cres); err != nil {
		h.log.Error("could not encode response body", zap.Error(err))
	}
}
