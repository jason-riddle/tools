.PHONY: build vet test clean

build:
	go build -o goober-app ./cmd/goober

vet:
	go vet ./...

test:
	go test ./...

clean:
	go clean ./...
