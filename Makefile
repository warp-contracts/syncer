PACKAGE  = syncer
GOROOT   = $(CURDIR)/.gopath~
GOPATH   = $(CURDIR)/.gopath~
BIN      = $(GOPATH)/bin
BASE     = $(GOPATH)/src/$(PACKAGE)
PATH    := bin:$(PATH)
GO       = go

export GOPATH

# Display utils
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

# Default target
.PHONY: all
all:  build lint | $(BASE); $(info $(M) built and lint everything!) @

# Setup
$(BASE): ; $(info $(M) setting GOPATH…)
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

# External tools 
$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) installing $(REPOSITORY)…)
	$Q tmp=$$(mktemp -d); \
	   env GO111MODULE=on GOPATH=$$tmp GOBIN=$(BIN) $(GO) install $(REPOSITORY) \
		|| ret=$$?; \
	   exit $$ret

GOLANGCILINT = $(BIN)/golangci-lint
$(BIN)/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.50.1

GENTOOL = $(BIN)/gentool
$(BIN)/gentool: REPOSITORY=gorm.io/gen/tools/gentool@latest

# Build targets
.PHONY: build
build:  | $(BASE); $(info $(M) building executable…) @
	$Q cd $(BASE) && $(GO) build \
		-tags release \
		-ldflags="-s -w  -X $(PACKAGE)/src/utils/build_info.Version=$(VERSION) -X $(PACKAGE)/src/utils/build_info.BuildDate=$(DATE)" \
		-o bin/$(PACKAGE) main.go

.PHONY: run
run: all | ; $(info $(M) starting app with default params…)
	bin/$(PACKAGE) sync

.PHONY: lint
lint: $(GOLANGCILINT) | $(BASE) ; $(info $(M) running golangci-lint) @
	$Q $(GOLANGCILINT) run 

.PHONY: generate
generate: $(GENTOOL) | $(BASE) ; $(info $(M) generating model from the database) @
	$Q $(GENTOOL) -db postgres -dsn "host=127.0.0.1 user=postgres password=postgres dbname=warp port=7654 sslmode=disable" -tables "interactions"

.PHONY: test
test:
	$(GO) test ./...

.PHONY: clean
clean:
	rm -rf bin/$(PACKAGE) .gopath~

.PHONY: docker-build
docker-build: all | ; $(info $(M) building docker container) @ 
	$(GO) mod vendor
	DOCKER_BUILDKIT=0 docker build -t "warp.cc/syncer:latest" .
	rm -rf vendor

.PHONY: docker-run
docker-run: docker-build | ; $(info $(M) running docker container) @ 
	docker compose --profile syncer up syncer