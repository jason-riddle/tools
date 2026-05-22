.PHONY: build build-all fmt vet test clean

build:
	go build -o goober-app ./cmd/goober

build-all:
	go build ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

test:
	go test ./...

clean:
	go clean ./...
	rm -f goober-app uuid-app
