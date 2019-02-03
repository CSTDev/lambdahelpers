GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
p=...


test:
	GO111MODULE=on $(GOTEST) -v ./pkg/$(p)

	