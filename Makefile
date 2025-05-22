# On Linux, every command that's executed with $(GOFER) is executed in a
# bubblewrap container without network access and with limited access to the
# filesystem.

OUTPUT ?= $(shell pwd)/build/
OUTPUT_RELEASE ?= $(OUTPUT)release/

MODULE := $(shell grep '^module' go.mod|cut -d' ' -f2)

SRC := main.go ./src ./tools
GOFER := go run main.go

export CGO_ENABLED := 0
export GO111MODULE := on
export GOFLAGS := -mod=readonly
export GOSUMDB := sum.golang.org
export REAL_GOPROXY := $(shell go env GOPROXY)
export GOPROXY := off

# Unfortunately there is no Go-specific way of pinning the CA for GOPROXY.
# The go.pem file is created by the `pin` target in this Makefile.
export SSL_CERT_FILE := ./go.pem
export SSL_CERT_DIR := /path/does/not/exist/to/pin/ca

define PIN_EXPLANATION
# The checksums for go.sum and go.mod are pinned because `go mod` with
# `-mod=readonly` isn't read-only.  The `go mod` commands will still modify the
# dependency tree if they find it necessary (e.g., to add a missing module or
# module checksum).
#
# Run `make pin` to update this file.
endef
export PIN_EXPLANATION

all: build

tidy:
	@GOPROXY=$(REAL_GOPROXY) go mod tidy
	@$(GOFER) run -- go mod verify

prepare-offline: tidy
	@GOPROXY=$(REAL_GOPROXY) go list -m -json all >/dev/null

build:
	@$(GOFER) build -o $(OUTPUT)

release:
	@$(GOFER) run -- go clean
	@$(GOFER) run -- go clean -cache
	@$(GOFER) run -- rm -rfv $(OUTPUT_RELEASE)
	@$(GOFER) build -o $(OUTPUT_RELEASE) --release \
		-t linux:amd64 \
		-t linux:arm64 \
		-t darwin:arm64 \
		-t windows:amd64

clean:
	@$(GOFER) run -- go clean
	@$(GOFER) run -- go clean -cache
	@$(GOFER) run -- rm -rfv $(OUTPUT)

distclean:
	@$(GOFER) run -- git clean -d -f -x

test:
	@$(GOFER) run -- mkdir -p $(OUTPUT)
	@$(GOFER) run -- go test -v -coverprofile=$(OUTPUT)/.coverage -coverpkg=./... ./...

coverage:
	@$(GOFER) run -- go tool cover -func $(OUTPUT)/.coverage

check-nilerr:
	@$(GOFER) run -- echo "Running nilerr"
	@$(GOFER) run -- nilerr ./...

check-errcheck:
	@$(GOFER) run -- echo "Running errcheck"
	@$(GOFER) run -- errcheck ./...

check-revive:
	@$(GOFER) run -- echo "Running revive"
	@$(GOFER) run -- revive -config revive.toml -set_exit_status ./...

check-gosec:
	@$(GOFER) run -- echo "Running gosec"
	@$(GOFER) run -- gosec -quiet ./...

check-staticcheck:
	@$(GOFER) run -- echo "Running staticcheck"
	@$(GOFER) run -- staticcheck ./...

check-vet:
	@$(GOFER) run -- echo "Running go vet"
	@$(GOFER) run -- go vet ./...

check-fmt:
	@$(GOFER) run -- echo "Running gofmt"
	@$(GOFER) run -- gofmt -d -l $(SRC)

check-imports:
	@$(GOFER) run -- echo "Running goimports"
	@$(GOFER) run -- goimports -d -local $(MODULE) -l $(SRC)

check: verify check-nilerr check-errcheck check-revive check-gosec check-staticcheck check-vet check-fmt check-imports

fix-fmt:
	@$(GOFER) run -- gofmt -w -l $(SRC)

fix-imports:
	@$(GOFER) run -- goimports -w -l -local $(MODULE) $(SRC)

fix: verify fix-fmt fix-imports

pin:
	@$(GOFER) run -- echo "$$PIN_EXPLANATION" > go.pin 2>&1
	@$(GOFER) run -- sha256sum go.sum go.mod >> go.pin 2>&1
	@test -f /etc/ssl/certs/GTS_Root_R1.pem && test -f /etc/ssl/certs/GTS_Root_R4.pem && \
		cat /etc/ssl/certs/GTS_Root_R1.pem /etc/ssl/certs/GTS_Root_R4.pem > go.pem || true

verify:
	@$(GOFER) run -- sha256sum --strict --check go.pin
	@$(GOFER) run -- go mod verify

qa: build check test coverage

.PHONY: all tidy build release clean distclean
.PHONY: test coverage prepare-offline
.PHONY: check-nilerr check-errcheck check-revive check-gosec check-staticcheck check-vet check-fmt check-imports check
.PHONY: fix-imports fix-fmt fix pin verify qa
