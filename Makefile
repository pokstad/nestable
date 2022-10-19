GOPATH=$(shell go env GOPATH)

$(GOPATH)/bin/nst: $(shell find . -type f -name "*.go")
	go install ./cmd/nst
