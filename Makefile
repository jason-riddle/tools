.PHONY: build build-all fmt vet test clean

build:
	go build -o gob-app ./cmd/gob

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
	rm -f gob-app uuid-app
