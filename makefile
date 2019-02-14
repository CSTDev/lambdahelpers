GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
p=...


test:
	GO111MODULE=on $(GOTEST) -v ./pkg/$(p)

build:
	GO111MODULE=on $(GOBUILD) -v ./...

benchmark:
	GO111MODULE=on $(GOTEST) -run=XXX -bench=. -test.count=10 -test.benchmem=true ./...
