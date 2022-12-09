PACKAGE  = syncer
GOPATH   = $(CURDIR)/.gopath~
BIN      = $(GOPATH)/bin
BASE     = $(GOPATH)/src/$(PACKAGE)
PATH    := bin:$(PATH)
GO       = go

# Display utils
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m▶\033[0m")

# Default target
.PHONY: all
all:  build | $(BASE); $(info $(M) built everything!) @

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


# Build targets
.PHONY: build
build:  | $(BASE); $(info $(M) building executable…) @
	$Q cd $(BASE) && $(GO) build \
		-tags release \
		-ldflags="-s -w  -X $(PACKAGE)/src/utils/build_info.Version=$(VERSION) -X $(PACKAGE)/src/utils/build_info.BuildDate=$(DATE)" \
		-o bin/$(PACKAGE) main.go

.PHONY: run
run: all | ; $(info $(M) starting app with default params…)
	bin/$(PACKAGE) server

.PHONY: lint
lint: $(GOLANGCILINT) | $(BASE) ; $(info $(M) running golangci-lint) @
	$Q $(GOLANGCILINT) run 

.PHONY: test
test:
	$(GO) test ./...

