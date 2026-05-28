.PHONY: build build-all fmt vet test nix-build nix-build-all nix-check clean

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

nix-build:
	nix build 'path:.#default'

nix-build-all:
	nix build 'path:.#gob'
	nix build 'path:.#uuid'

nix-check: build-all vet test nix-build-all

clean:
	go clean ./...
	rm -f gob-app uuid-app
