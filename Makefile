SHELL=/usr/bin/env bash

export CGO_CFLAGS_ALLOW=-D__BLST_PORTABLE__
export CGO_CFLAGS=-D__BLST_PORTABLE__

GOVERSION:=$(shell go version | cut -d' ' -f 3 | cut -d. -f 2)
ifeq ($(shell expr $(GOVERSION) \< 13), 1)
$(warning Your Golang version is go 1.$(GOVERSION))
$(error Update Golang to version $(shell grep '^go' go.mod))
endif

CLEAN:=
BINS:=./venus-gateway

git=$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))

ldflags=-X=github.com/ipfs-force-community/venus-gateway/version.CurrentCommit='+git$(git)'
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"


## FFI

FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): extern/.filecoin-install ;

extern/.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=extern/.filecoin-install
CLEAN+=extern/.filecoin-install

ffi-version-check:
	@[[ "$$(awk '/const Version/{print $$5}' extern/filecoin-ffi/version.go)" -eq 3 ]] || (echo "FFI version mismatch, update submodules"; exit 1)
BUILD_DEPS+=ffi-version-check

.PHONY: ffi-version-check

$(MODULES): extern/.update-modules ;
# dummy file that marks the last time modules were updated
extern/.update-modules:
	git submodule update --init --recursive
	touch $@

gateway: show-env $(BUILD_DEPS)
	rm -f venus-gateway
	go build $(GOFLAGS) -o venus-gateway
	./venus-gateway --version


linux: 	show-env $(BUILD_DEPS)
	rm -f venus-gateway
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc CGO_LDFLAGS="-static" go build $(GOFLAGS) -o venus-gateway

show-env:
	@echo '_________________build_environment_______________'
	@echo '| CC=$(CC)'
	@echo '| CGO_CFLAGS=$(CGO_CFLAGS)'
	@echo '| git commit=$(git)'
	@echo '-------------------------------------------------'

lint:
	gofmt -s -w ./
	golangci-lint run

clean:
	rm -rf $(CLEAN) $(BINS)
.PHONY: clean

print-%:
	@echo $*=$($*)
