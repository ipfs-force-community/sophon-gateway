export CGO_CFLAGS_ALLOW=-D__BLST_PORTABLE__
export CGO_CFLAGS=-D__BLST_PORTABLE__

all: build
.PHONY: all

## variables

# git modules that need to be loaded
MODULES:=

ldflags=-X=github.com/ipfs-force-community/sophon-gateway/version.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	    ldflags+=-extldflags=$(LDFLAGS)
	endif

GOFLAGS+=-ldflags="$(ldflags)"

## FFI

FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): build-dep/.filecoin-install ;

build-dep/.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=build-dep/.filecoin-install
CLEAN+=build-dep/.filecoin-install

## modules
build-dep:
	mkdir $@

$(MODULES): build-dep/.update-modules;
# dummy file that marks the last time modules were updated
build-dep/.update-modules: build-dep;
	git submodule update --init --recursive
	touch $@

## build

test:
	go test -race ./...

lint: $(BUILD_DEPS)
	golangci-lint run

deps: $(BUILD_DEPS)

dist-clean:
	git clean -xdff
	git submodule deinit --all -f

build: $(BUILD_DEPS)
	rm -f sophon-gateway
	go build -o ./sophon-gateway $(GOFLAGS) .

debug: $(BUILD_DEPS)
	rm -f sophon-gateway
	go build -gcflags=all="-N -l" -o ./sophon-gateway $(GOFLAGS) .

.PHONY: docker

TAG:=test
docker: $(BUILD_DEPS)
ifdef DOCKERFILE
	cp $(DOCKERFILE) ./dockerfile
else
	curl -o dockerfile https://raw.githubusercontent.com/filecoin-project/venus-docs/master/script/docker/dockerfile
endif
	docker build --build-arg HTTPS_PROXY=$(BUILD_DOCKER_PROXY) --build-arg BUILD_TARGET=sophon-gateway -t sophon-gateway .
	docker tag sophon-gateway filvenus/sophon-gateway:$(TAG)
ifdef PRIVATE_REGISTRY
	docker tag sophon-gateway $(PRIVATE_REGISTRY)/filvenus/sophon-gateway:$(TAG)
endif

docker-push: docker
ifdef PRIVATE_REGISTRY
	docker push $(PRIVATE_REGISTRY)/filvenus/sophon-gateway:$(TAG)
else
	docker push filvenus/sophon-gateway:$(TAG)
	docker tag filvenus/sophon-gateway:$(TAG) filvenus/sophon-gateway:latest
	docker push filvenus/sophon-gateway:latest
endif
