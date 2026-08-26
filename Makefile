GO=GOTOOLCHAIN=local go

test:
	$(GO) test ./... -count=1

race:
	$(GO) test -race ./... -count=1

vet:
	$(GO) vet ./...

build:
	$(GO) build ./...

run:
	$(GO) run ./cmd/server
