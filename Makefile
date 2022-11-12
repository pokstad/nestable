GOPATH=$(shell go env GOPATH)
BUILD_TAGS=sqlite_fts5

$(GOPATH)/bin/nst: $(shell find . -type f -name "*.go")
	go install -tags $(BUILD_TAGS) ./cmd/nst

.PHONY: test
test:
	go test -v -tags $(BUILD_TAGS) ./...

.PHONY: build
build:
	go build -v -tags $(BUILD_TAGS) ./...

.PHONY: vet
vet:
	go vet -tags $(BUILD_TAGS) ./...
