.PHONY: build test vet parity latency

BIN := bin/statusline

build:
	@mkdir -p bin
	go build -o $(BIN) ./cmd/statusline

vet:
	go vet ./...

test:
	go test -race ./...

# The parity oracle: the new binary against the legacy goldens.
parity: build
	cd testdata/parity/harness && go run . check -bin ../../../$(BIN) -flavour kit

latency: build
	cd testdata/parity/harness && go run . latency -bin ../../../$(BIN) -flavour kit -runs 300 -scenario busy/default
	cd testdata/parity/harness && go run . latency -bin ../../../$(BIN) -flavour kit -cold -runs 100 -scenario busy/default
