GO_FILES := $(shell git ls-files '*.go')
COVER_FLOOR := 40

.PHONY: build test vet fmt staticcheck cover check tools

build:
	go build ./...

test:
	go test ./... -race

vet:
	go vet ./...

# Fail if any tracked Go file is not gofmt-clean.
fmt:
	@unformatted=$$(gofmt -l $(GO_FILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed:"; echo "$$unformatted"; exit 1; \
	fi

staticcheck:
	staticcheck ./...

# Run tests with coverage and enforce the floor.
cover:
	@go test ./... -coverprofile=coverage.out >/dev/null
	@pct=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
	echo "total coverage: $$pct% (floor $(COVER_FLOOR)%)"; \
	awk "BEGIN{exit !($$pct >= $(COVER_FLOOR))}" || { echo "coverage below floor"; exit 1; }

# Install the lint tool used by the gate (kept out of the module).
tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest

# The full deterministic gate, shared by local dev and CI.
check: fmt vet staticcheck build cover
