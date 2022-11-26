GOPATH=$(shell go env GOPATH)
BUILD_TAGS=sqlite_fts5

VUE=internal/nestable-vue

$(GOPATH)/bin/nst: $(shell find . -type f) internal/web/static
	go install -tags $(BUILD_TAGS) ./cmd/nst

$(VUE)/node_modules: $(VUE)/package.json
	cd $(VUE) && npm install

$(VUE)/dist: $(VUE)/index.html $(VUE)/node_modules/* $(VUE)/public $(VUE)/src
	cd $(VUE) && npm run build

internal/web/static: $(VUE)/dist
	rm -rf internal/web/static/*
	cp -R $(VUE)/dist/* internal/web/static/

.PHONY: test
test:
	go test -v -tags $(BUILD_TAGS) ./...

.PHONY: build
build:
	go build -v -tags $(BUILD_TAGS) ./...

.PHONY: vet
vet:
	go vet -tags $(BUILD_TAGS) ./...
