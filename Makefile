BIN := hypr-layout
PKG := ./cmd/hypr-layout

build:
	go build -o $(BIN) $(PKG)

run:
	go run $(PKG)

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

check: fmt vet test

clean:
	rm -f $(BIN)

.PHONY: build run fmt vet test check clean
