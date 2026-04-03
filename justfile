build:
    go build -o bin/axon-code ./cmd/axon-code

install: build
    cp bin/axon-code ~/.local/bin/axon-code

test:
    go test ./...

vet:
    go vet ./...
