.PHONY: build test vet parity latency design

BIN := bin/statusline

# VERSION and VENDOR_KEY (base64 ed25519) make a release build (design/: binaries[].build); without
# them the binary is a development build that never updates itself.
LDFLAGS := -X main.version=$(VERSION) -X main.vendorKey=$(VENDOR_KEY)

# A static binary: no cgo resolver or user lookup, no dynamic loader at every
# start of the client (~1 ms at p50).
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/statusline

# The structure against design/: regenerate at blank, then kit's check.
design:
	kit gen -check
	kit check
	kit test

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
