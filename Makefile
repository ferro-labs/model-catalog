.PHONY: build test validate lint split clean fmt

FERROCAT = go run ./cmd/ferrocat

build:
	$(FERROCAT) build --output dist/

test:
	go test -race -count=1 ./...

validate:
	$(FERROCAT) validate

lint:
	golangci-lint run ./...

split:
	@test -n "$(INPUT)" || (echo "Usage: make split INPUT=/path/to/catalog.json" && exit 1)
	$(FERROCAT) split $(INPUT) --output providers/

clean:
	rm -rf dist/

fmt:
	gofmt -w .
