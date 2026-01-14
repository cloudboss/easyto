PROJECT = $(shell basename ${PWD})
OS = $(shell uname | tr [:upper:] [:lower:])
ARCH = $(shell arch=$$(uname -m); [ "$${arch}" = "x86_64" ] && echo "amd64" || echo $${arch})
VERSION =
DIR_OUT = _output

DIR_STG = $(DIR_OUT)/staging
DIR_STG_EASYTO = $(DIR_STG)/easyto/$(OS)/$(ARCH)
DIR_STG_ASSETS = $(DIR_STG_EASYTO)/assets
DIR_STG_BIN = $(DIR_STG_EASYTO)/bin
DIR_STG_INIT = $(DIR_STG)/init
DIR_STG_PACKER = $(DIR_STG_EASYTO)/packer
DIR_STG_PACKER_PLUGIN = $(DIR_STG_PACKER)/plugins/github.com/hashicorp/amazon
DIRS_STG_CLEAN = $(DIR_STG_INIT)

ifneq ($(MAKECMDGOALS), clean)
include $(DIR_OUT)/Makefile.inc
endif

DIR_RELEASE = $(DIR_OUT)/release

EASYTO_ASSETS_RELEASES = https://github.com/cloudboss/easyto-assets/releases/download
EASYTO_ASSETS_VERSION = v0.5.1
EASYTO_ASSETS_BUILD = easyto-assets-build-$(EASYTO_ASSETS_VERSION)
EASYTO_ASSETS_BUILD_ARCHIVE = $(EASYTO_ASSETS_BUILD).tar.gz
EASYTO_ASSETS_BUILD_URL = $(EASYTO_ASSETS_RELEASES)/$(EASYTO_ASSETS_VERSION)/$(EASYTO_ASSETS_BUILD_ARCHIVE)
EASYTO_ASSETS_PACKER = easyto-assets-packer-$(EASYTO_ASSETS_VERSION)-$(OS)-$(ARCH)
EASYTO_ASSETS_PACKER_ARCHIVE = $(EASYTO_ASSETS_PACKER).tar.gz
EASYTO_ASSETS_PACKER_URL = $(EASYTO_ASSETS_RELEASES)/$(EASYTO_ASSETS_VERSION)/$(EASYTO_ASSETS_PACKER_ARCHIVE)
EASYTO_ASSETS_RUNTIME = easyto-assets-runtime-$(EASYTO_ASSETS_VERSION)
EASYTO_ASSETS_RUNTIME_ARCHIVE = $(EASYTO_ASSETS_RUNTIME).tar.gz
EASYTO_ASSETS_RUNTIME_URL = $(EASYTO_ASSETS_RELEASES)/$(EASYTO_ASSETS_VERSION)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE)
EASYTO_INIT_RELEASES = https://github.com/cloudboss/easyto-init/releases/download
EASYTO_INIT_VERSION = v0.3.0
EASYTO_INIT = easyto-init-$(EASYTO_INIT_VERSION)
EASYTO_INIT_ARCHIVE = easyto-init-$(EASYTO_INIT_VERSION).tar.gz
EASYTO_INIT_URL = $(EASYTO_INIT_RELEASES)/$(EASYTO_INIT_VERSION)/$(EASYTO_INIT_ARCHIVE)

EASYTO_ASSETS_PACKER_OUT = $(DIR_STG_PACKER)/$(PACKER_EXE) \
	$(DIR_STG_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_EXE) \
	$(DIR_STG_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_EXE)_SHA256SUM

EASYTO_ASSETS_RUNTIME_OUT = $(DIR_STG_ASSETS)/base.tar \
	$(DIR_STG_ASSETS)/boot.tar \
	$(DIR_STG_ASSETS)/chrony.tar \
	$(DIR_STG_ASSETS)/kernel.tar \
	$(DIR_STG_ASSETS)/ssh.tar

.DEFAULT_GOAL = release

FORCE:

$(DIR_OUT):
	@mkdir -p $(DIR_OUT)

$(DIR_OUT)/Makefile.inc: FORCE $(DIR_OUT)/$(EASYTO_ASSETS_BUILD_ARCHIVE)
	@tar -zx --xform "s|^$(EASYTO_ASSETS_BUILD)/./|$(DIR_OUT)/tmp-|" \
		-f $(DIR_OUT)/$(EASYTO_ASSETS_BUILD_ARCHIVE) \
		$(EASYTO_ASSETS_BUILD)/./Makefile.inc
	@cmp -s $(DIR_OUT)/tmp-Makefile.inc $(DIR_OUT)/Makefile.inc 2>/dev/null && \
		rm -f $(DIR_OUT)/tmp-Makefile.inc || \
		mv $(DIR_OUT)/tmp-Makefile.inc $(DIR_OUT)/Makefile.inc

$(DIR_OUT)/$(EASYTO_ASSETS_BUILD_ARCHIVE): | $(HAS_COMMAND_CURL) $(DIR_OUT)
	@curl -L -o $(DIR_OUT)/$(EASYTO_ASSETS_BUILD_ARCHIVE) $(EASYTO_ASSETS_BUILD_URL)

$(DIR_OUT)/$(EASYTO_ASSETS_PACKER_ARCHIVE): | $(HAS_COMMAND_CURL) $(DIR_OUT)
	@curl -L -o $(DIR_OUT)/$(EASYTO_ASSETS_PACKER_ARCHIVE) $(EASYTO_ASSETS_PACKER_URL)

$(DIR_OUT)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE): | $(HAS_COMMAND_CURL) $(DIR_OUT)
	@curl -L -o $(DIR_OUT)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE) $(EASYTO_ASSETS_RUNTIME_URL)

$(DIR_OUT)/$(EASYTO_INIT_ARCHIVE): | $(HAS_COMMAND_CURL) $(DIR_OUT)
	@curl -L -o $(DIR_OUT)/$(EASYTO_INIT_ARCHIVE) $(EASYTO_INIT_URL)

$(EASYTO_ASSETS_PACKER_OUT) &: $(DIR_OUT)/$(EASYTO_ASSETS_PACKER_ARCHIVE) | $(DIR_STG_PACKER)/
	@tar -zmx \
		--xform "s|^$(EASYTO_ASSETS_PACKER)|$(DIR_STG_PACKER)|" \
		-f $(DIR_OUT)/$(EASYTO_ASSETS_PACKER_ARCHIVE)

$(DIR_STG_PACKER)/build.pkr.hcl: $(DIR_ROOT)/packer/build.pkr.hcl | $(DIR_STG_PACKER)/
	@install -m 0644 $(DIR_ROOT)/packer/build.pkr.hcl $(DIR_STG_PACKER)/build.pkr.hcl

$(DIR_STG_PACKER)/provision: $(DIR_ROOT)/packer/provision
	@install -m 0755 $(DIR_ROOT)/packer/provision $(DIR_STG_PACKER)/provision

$(EASYTO_ASSETS_RUNTIME_OUT) &: $(DIR_OUT)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE) | $(DIR_STG_ASSETS)/
	@tar -zmx \
		--xform "s|^$(EASYTO_ASSETS_RUNTIME)|$(DIR_STG_ASSETS)|" \
		-f $(DIR_OUT)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE)

$(DIR_STG_ASSETS)/ctr2disk: $(DIR_OUT)/ctr2disk | $(DIR_STG_ASSETS)/
	@install -m 0755 $(DIR_OUT)/ctr2disk $(DIR_STG_ASSETS)/ctr2disk

$(DIR_OUT)/mke2fs-tmp/base.tar: $(DIR_OUT)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE) | $(DIR_OUT)/mke2fs-tmp/
	@tar -zxf $(DIR_OUT)/$(EASYTO_ASSETS_RUNTIME_ARCHIVE) \
		--wildcards \
		--xform "s|^$(EASYTO_ASSETS_RUNTIME).*/||" \
		-C $(DIR_OUT)/mke2fs-tmp \
		$(EASYTO_ASSETS_RUNTIME)/./base.tar
	@touch $(DIR_OUT)/mke2fs-tmp/base.tar

$(DIR_OUT)/mke2fs-tmp/mke2fs: $(DIR_OUT)/mke2fs-tmp/base.tar
	@tar -C $(DIR_OUT)/mke2fs-tmp \
		-xf $(DIR_OUT)/mke2fs-tmp/base.tar \
		--wildcards \
		--xform "s|.*/||" \
		'*/mke2fs' \
		'*/mkfs.ext*'
	@touch $(DIR_OUT)/mke2fs-tmp/mke2fs

embed/mke2fs.bin: $(DIR_OUT)/mke2fs-tmp/mke2fs
	@cp $(DIR_OUT)/mke2fs-tmp/mke2fs embed/mke2fs.bin

$(DIR_OUT)/ctr2disk: \
		hack/compile-ctr2disk-ctr \
		go.mod \
		embed/mke2fs.bin \
		$(shell find cmd/ctr2disk -type f -path '*.go' ! -path '*_test.go') \
		$(shell find pkg -type f -path '*.go' ! -path '*_test.go') \
		| $(HAS_IMAGE_LOCAL) $(VAR_DIR_ET)
	@docker run --rm -t \
		-v $(DIR_ROOT):/code \
		-e OPENSSH_PRIVSEP_DIR=$(OPENSSH_PRIVSEP_DIR) \
		-e OPENSSH_PRIVSEP_USER=$(OPENSSH_PRIVSEP_USER) \
		-e CHRONY_USER=$(CHRONY_USER) \
		-e DIR_ET_ROOT=/$(DIR_ET) \
		-e DIR_OUT=/code/$(DIR_OUT) \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-ctr2disk-ctr)"

$(DIR_STG_ASSETS)/init.tar: $(DIR_OUT)/$(EASYTO_INIT_ARCHIVE) | $(DIR_STG_ASSETS)/
	@tar -zmx \
		--xform "s|^$(EASYTO_INIT)|$(DIR_STG_ASSETS)|" \
		-f $(DIR_OUT)/$(EASYTO_INIT_ARCHIVE)

$(DIR_STG_BIN)/easyto: \
		hack/compile-easyto-ctr \
		go.mod \
		$(shell find cmd/easyto -type f -path '*.go' ! -path '*_test.go') \
		| $(HAS_IMAGE_LOCAL) $(VAR_DIR_ET) $(DIR_STG_BIN)/
	@docker run --rm -t \
		-v $(DIR_ROOT):/code \
		-e DIR_ET_ROOT=/$(DIR_ET) \
		-e DIR_OUT=/code/$(DIR_STG_BIN) \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOARCH=$(ARCH) \
		-e GOOS=$(OS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-easyto-ctr)"

$(DIR_RELEASE)/easyto-$(VERSION)-$(OS)-$(ARCH).tar.gz: \
		$(EASYTO_ASSETS_PACKER_OUT) \
		$(DIR_STG_PACKER)/build.pkr.hcl \
		$(DIR_STG_PACKER)/provision \
		$(EASYTO_ASSETS_RUNTIME_OUT) \
		$(DIR_STG_ASSETS)/ctr2disk \
		$(DIR_STG_ASSETS)/init.tar \
		$(DIR_STG_BIN)/easyto \
		| $(HAS_COMMAND_FAKEROOT) $(DIR_RELEASE)/
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@[ $$(echo $(VERSION) | cut -c 1) = v ] || (echo "VERSION must begin with a 'v'"; exit 1)
	@cd $(DIR_STG_EASYTO) && \
		fakeroot tar -cz \
		--xform "s|^|easyto-$(VERSION)/|" \
		-f $(DIR_ROOT)/$(DIR_RELEASE)/easyto-$(VERSION)-$(OS)-$(ARCH).tar.gz assets bin packer

test: embed/mke2fs.bin
	go vet -v ./...
	go test -v ./...

release-one: $(DIR_RELEASE)/easyto-$(VERSION)-$(OS)-$(ARCH).tar.gz

release-linux-%:
	$(MAKE) OS=linux ARCH=$* VERSION=$(VERSION) release-one

release-darwin-%:
	$(MAKE) OS=darwin ARCH=$* VERSION=$(VERSION) release-one

release-windows-%:
	$(MAKE) OS=windows ARCH=$* VERSION=$(VERSION) release-one

release: release-linux-amd64 release-linux-arm64 \
	release-darwin-amd64 release-darwin-arm64 \
	release-windows-amd64

clean:
	@chmod -R +w $(DIR_OUT)/go
	@rm -rf $(DIR_OUT)

.PHONY: test release-one \
	release-linux-amd64 release-linux-arm64 \
	release-darwin-amd64 release-darwin-arm64 \
	release-windows-amd64 release clean
