.PHONY: build run clean test lint

build:
	go build -o otlp9s ./cmd/otlp9s/

run: build
	./otlp9s --forward localhost:4320

test:
	go test -race -v ./...

lint:
	go vet ./...

clean:
	rm -f otlp9s
