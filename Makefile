APP      := prj
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X github.com/gorodulin/prj/cmd.version=$(VERSION)
DIST     := dist

PLATFORMS := darwin/amd64 darwin/arm64 \
             linux/amd64 linux/arm64 \
             windows/amd64 windows/arm64 \
             freebsd/amd64

.PHONY: help build test lint check cover clean install cross release

help: ## Show available targets
	@grep -E '^[a-z][a-z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk -F ':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

build: ## Build binary
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP) .

test: ## Run tests with race detector and coverage
	go test -v -race -cover ./...

lint: ## Run static analysis (go vet + staticcheck)
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

check: test lint ## Run tests then lint (CI target)

cover: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

clean: ## Remove build artifacts
	rm -f $(APP) coverage.out coverage.html
	rm -rf $(DIST)

install: ## Install to $GOPATH/bin
	go install -trimpath -ldflags "$(LDFLAGS)" .

release: ## Tag, release, and update packaging (V=x.y.z [DRY_RUN=1])
	@test -n "$(V)" || { echo "Current: $$(git describe --tags --abbrev=0 2>/dev/null || echo none)"; echo "Usage: make release V=x.y.z [DRY_RUN=1]"; exit 1; }
	@scripts/release.sh $(V) $(if $(DRY_RUN),--dry-run)

cross: ## Cross-compile for all platforms
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} CGO_ENABLED=0 \
		go build -trimpath -ldflags "$(LDFLAGS)" \
			-o $(DIST)/$(APP)-$${platform%/*}-$${platform#*/}$$([ "$${platform%/*}" = "windows" ] && echo .exe) . ; \
		echo "  built $(DIST)/$(APP)-$${platform%/*}-$${platform#*/}" ; \
	done
