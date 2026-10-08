# nilemu — NilOS Studio Emulator
#
# Everyday targets:
#   make run       start the emulator (http://127.0.0.1:8085/studio.html)
#   make test      run the full test suite
#   make build     build all binaries into ./bin
#   make tools     build nilctl / nilosd / nilsign into ./bin

GO      ?= go
BIN     ?= bin
LDFLAGS ?= -s -w

.PHONY: all build tools run test vet fmt clean release

all: build

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nilemu .
	$(MAKE) tools

tools:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nilosd ./cmd/nilosd
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nilctl ./cmd/nilctl
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nilsign ./cmd/nilsign
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nilsp ./cmd/nilsp
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nildap ./cmd/nildap
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN)/nilstudio ./cmd/nilstudio

run:
	$(GO) run .

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

clean:
	$(GO) clean
	rm -rf $(BIN)

release: build
	@echo "Binaries in $(BIN)/"
	@ls -lh $(BIN)
