VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILDINFO_PACKAGE := github.com/vwall/kitout/internal/buildinfo
LDFLAGS := -s -w -X $(BUILDINFO_PACKAGE).Version=$(VERSION) -X $(BUILDINFO_PACKAGE).Commit=$(COMMIT) -X $(BUILDINFO_PACKAGE).BuildDate=$(BUILD_DATE)

.PHONY: build test vet smoke-distribution release-check

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/kitout ./cmd/kitout

test:
	go test ./...

vet:
	go vet ./...

smoke-distribution: build
	scripts/smoke-distribution.sh

release-check:
	$(MAKE) test
	$(MAKE) vet
	$(MAKE) smoke-distribution VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)"
