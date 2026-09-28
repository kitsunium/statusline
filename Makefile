.PHONY: build test vet parity latency design

BIN := bin/statusline

# VERSION and VENDOR_KEY (base64 ed25519) make a release build; without
# them the binary is a development build that never updates itself.
LDFLAGS := -X github.com/kitsunium/statusline/ipc.version=$(VERSION) -X github.com/kitsunium/statusline/collect/releases.vendorKey=$(VENDOR_KEY)

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/statusline

# The structure against design/: regenerate at blank, then kit's check.
design:
	kit gen -check
	kit check

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
