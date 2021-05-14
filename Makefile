GO         = go
GOGET      = $(GO) get -u
GOFLAGS   ?= -mod=vendor
GOPATH    ?= $(HOME)/go

OAPI_CODEGEN = $(GOPATH)/bin/oapi-codegen

GIN        = $(GOPATH)/bin/gin
GIN_ARG_PORT  ?= 3100
GIN_ARG_APORT ?= 3101
GIN_ARG_LADDR ?= localhost
GIN_ARGS      ?= --laddr $(GIN_ARG_LADDR) --port $(GIN_ARG_PORT) --appPort $(GIN_ARG_APORT) --immediate


CODEGEN_API = api/gen.go

watch: $(GIN)
	$(GIN) $(GIN_ARGS) run -- serve

clean:
	rm -f $(CODEGEN_API)

codegen: $(OAPI_CODEGEN)
	$(OAPI_CODEGEN) --config api/codegen-config.yaml -o $(CODEGEN_API) api/def.yaml

$(GIN):
	$(GOGET) github.com/codegangsta/gin

$(OAPI_CODEGEN):
	$(GOGET) github.com/deepmap/oapi-codegen/cmd/oapi-codegen
