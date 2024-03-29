SHELL := /bin/bash
EMPTY=
SPACE=$(EMPTY) $(EMPTY)
COMMA=,#


## version pkg list
PROJECT_NAME=$(shell grep '^module\s\+.\+$$' go.mod|sed 's/module[[:blank:]]*//')
HEADER_FILE_NAME := ./script/boilerplate/boilerplate.go.txt
VERSION_PACKAGES=k8s.io/client-go/pkg/version k8s.io/component-base/version
IMAGE_PREFIX ?= ghcr.io/sxllwx

# kubernetes versioned pkg:
# - https://github.com/kubernetes/component-base/blob/master/version/base.go
# - https://github.com/kubernetes/client-go/blob/master/pkg/version/base.go
# https://github.com/kubernetes/kubernetes/blob/master/hack/lib/version.sh#L69
define  ADD_VERSION
-X $(1).gitVersion=$(GIT_VERSION) \
-X $(1).gitCommit=$(GIT_COMMIT) \
-X $(1).gitTreeState=$(GIT_TREE_STATE) \
-X $(1).buildDate=$(BUILD_DATE)
endef


## read all applications
APPS = $(notdir $(wildcard ./cmd/*))

## golang setting
GO ?= go
## disable cgo ...
## static build binary
GO := CGO_ENABLED=0 $(GO)
## version config
GIT_COMMIT ?= $(shell git rev-parse HEAD)
GIT_VERSION ?= $(shell git describe --dirty --always --tags --abbrev=7 --match 'v*')
BUILD_DATE ?= $(shell date ${SOURCE_DATE_EPOCH:+"--date=@${SOURCE_DATE_EPOCH}"} -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_TREE_STATE = $(if $(shell git status --porcelain 2>/dev/null),dirty,clean)

# build ldflag
GO_LDFLAGS += $(foreach pkg, $(subst $(COMMA),$(SPACE),$(VERSION_PACKAGES)),$(call ADD_VERSION,$(pkg)))
GO_BUILD_FLAGS += -trimpath -ldflags '-s -w $(GO_LDFLAGS)'

# platform setting
ifndef PLATFORMS
# PLATFORM's naming follows these well-known tools
# - https://skaffold.dev/docs/builders/builder-types/custom/
# - https://docs.docker.com/build/building/multi-platform/
## we use local os and arch, 
PLATFORMS:=$(shell ${GO} env GOHOSTOS)/$(shell ${GO} env GOHOSTARCH)
endif
ifdef RELEASE
PUSH_IMAGE:=1
# space-separated strings are better consumed by bash and make.
PLATFORMS:=linux/amd64 linux/arm64
endif

OCI_BUILD_CMD ?= docker buildx build
ifdef PUSH_IMAGE
OCI_BUILD_CMD := $(OCI_BUILD_CMD) --push
endif

## prepare the workdir
CUR_DIR := $(shell pwd)
ROOT_DIR := $(CUR_DIR)
OUTPUT_DIR ?= $(CUR_DIR)/_output
$(shell mkdir -p $(OUTPUT_DIR))
TOOLS_DIR ?= $(OUTPUT_DIR)/tools
$(shell mkdir -p $(TOOLS_DIR))
BINS_DIR ?= $(OUTPUT_DIR)/bin
$(shell mkdir -p $(BINS_DIR))
IMAGES_DIR ?= $(OUTPUT_DIR)/images
$(shell mkdir -p $(IMAGES_DIR))

.PHONY: all
all: tools.goimports fmt build

fmt: tools.goimports
	@echo "=======> goimports ..."
	@find . -type f \
		-name '*.go' \
		! \( -path './pkg/client/*' \
		-o -path './pkg/informers/*'  \
		-o -path './pkg/openapi/*'  \
		-o -path './pkg/applyconfigurations/*'  \
		-o -path './pkg/listers/*' \)  \
		! -name '*generated*' \
		-print -exec $(TOOLS_DIR)/goimports -local $(PROJECT_NAME) -w {} \;

.PHONY: clean
clean: clean.generated
	@-rm -rf $(OUTPUT_DIR)
	@-rm -rf vendor

.PHONY: build
build: fmt $(foreach app,$(APPS),$(addprefix build., $(app)))

# expect target: build.$(app)
build.%:
	$(eval app:=$(word 1,$(subst ., ,$*)))
	@$(foreach platform,$(PLATFORMS), \
		echo "======> building $(app):$(GIT_VERSION) ($(platform)) ..."; \
		$(eval OS:=$(word 1,$(subst /, ,$(platform))))\
		$(eval ARCH:=$(word 2,$(subst /, ,$(platform))))\
		GOOS=$(OS) GOARCH=$(ARCH) $(GO) \
				build \
        		$(GO_BUILD_FLAGS) \
        		-o $(BINS_DIR)/$(platform)/ \
        		./cmd/$(app);)


# install toolset
.PHONY: install.tools
install.tools: tools.goimports tools.ginkgo tools.k8s-generators tools.protoc-gen-go tools.protoc-gen-gogo

.PHONY: tools.goimports
tools.goimports:
	@GOBIN=$(TOOLS_DIR) $(GO) install golang.org/x/tools/cmd/goimports@latest

.PHONY: image
image: $(foreach app,$(APPS),$(addprefix image.,$(app)))

# target will image.kubectl-foo | image.kubectl-bar ...
# https://docs.github.com/cn/packages/working-with-a-github-packages-registry/working-with-the-container-registry
image.%: build.%
	$(eval app:=$(word 1,$(subst ., ,$*)))
	$(eval image:=$(IMAGE_PREFIX)/$(app):$(GIT_VERSION))
	@$(foreach platform, $(PLATFORMS), \
		mkdir -p $(IMAGES_DIR)/$(app)/$(platform); \
		cp -f ./build/image/Dockerfile $(IMAGES_DIR)/$(app)/Dockerfile; \
		cp -f $(BINS_DIR)/$(platform)/$(app) $(IMAGES_DIR)/$(app)/$(platform); \
	)
	@$(OCI_BUILD_CMD) --platform $(subst $(SPACE),$(COMMA),$(PLATFORMS))  --build-arg app=$(app) -t $(image) $(IMAGES_DIR)/$(app)/