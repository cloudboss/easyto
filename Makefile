PROJECT := $(shell basename ${PWD})
OS := $(shell uname | tr [:upper:] [:lower:])
ARCH := $(shell arch=$$(uname -m); [ "$${arch}" = "x86_64" ] && echo "amd64" || echo $${arch})
VERSION :=

DIR_ROOT := $(shell echo ${PWD})
DIR_OUT := _output
DIR_CB := __cb__
DIR_BOOTLOADER_TMP := $(DIR_OUT)/bootloader-tmp
DIR_BOOTLOADER_STG := $(DIR_OUT)/staging/bootloader
DIR_PREINIT_STG := $(DIR_OUT)/staging/preinit
DIR_KERNEL_STG := $(DIR_OUT)/staging/kernel
DIR_CHRONY_STG := $(DIR_OUT)/staging/chrony
DIR_SSH_STG := $(DIR_OUT)/staging/ssh
DIR_RELEASE := $(DIR_OUT)/release/$(OS)/$(ARCH)
DIR_RELEASE_ASSETS := $(DIR_RELEASE)/assets
DIR_RELEASE_BIN := $(DIR_RELEASE)/bin
DIR_RELEASE_PACKER := $(DIR_RELEASE)/packer
DIR_RELEASE_PACKER_PLUGIN := $(DIR_RELEASE_PACKER)/plugins/github.com/hashicorp/amazon
DIR_OSARCH_BUILD := $(DIR_OUT)/osarch/$(OS)/$(ARCH)
DIR_OPENSSH_DEPS := openssh-deps

DOCKERFILE_SHA256 := $(shell sha256sum Dockerfile.build | awk '{print $$1}' | cut -c 1-40)
CTR_IMAGE_GO := golang:1.21.0-alpine3.18
CTR_IMAGE_LOCAL := $(PROJECT):$(DOCKERFILE_SHA256)

KERNEL_ORG := https://cdn.kernel.org/pub/linux

BTRFSPROGS_VERSION := 6.5.2
BTRFSPROGS_SRC := btrfs-progs-v$(BTRFSPROGS_VERSION)
BTRFSPROGS_ARCHIVE := $(BTRFSPROGS_SRC).tar.xz
BTRFSPROGS_URL := $(KERNEL_ORG)/kernel/people/kdave/btrfs-progs/$(BTRFSPROGS_ARCHIVE)

E2FSPROGS_VERSION := 1.47.0
E2FSPROGS_SRC := e2fsprogs-$(E2FSPROGS_VERSION)
E2FSPROGS_ARCHIVE := $(E2FSPROGS_SRC).tar.gz
E2FSPROGS_URL := $(KERNEL_ORG)/kernel/people/tytso/e2fsprogs/v$(E2FSPROGS_VERSION)/$(E2FSPROGS_ARCHIVE)

KERNEL_VERSION := 6.1.52
KERNEL_VERSION_MAJ := $(shell echo $(KERNEL_VERSION) | cut -c 1)
KERNEL_SRC := linux-$(KERNEL_VERSION)
KERNEL_ARCHIVE := $(KERNEL_SRC).tar.xz
KERNEL_URL := $(KERNEL_ORG)/kernel/v$(KERNEL_VERSION_MAJ).x/$(KERNEL_ARCHIVE)

PACKER_VERSION := 1.9.4
PACKER_ARCHIVE := packer_$(PACKER_VERSION)_$(OS)_$(ARCH).zip
PACKER_URL := https://releases.hashicorp.com/packer/$(PACKER_VERSION)/$(PACKER_ARCHIVE)

PACKER_PLUGIN_AMZ_VERSION := 1.2.6
PACKER_PLUGIN_AMZ_FILE := packer-plugin-amazon_v$(PACKER_PLUGIN_AMZ_VERSION)_x5.0_$(OS)_$(ARCH)
PACKER_PLUGIN_AMZ_ARCHIVE := $(PACKER_PLUGIN_AMZ_FILE).zip
PACKER_PLUGIN_AMZ_URL := https://github.com/hashicorp/packer-plugin-amazon/releases/download/v$(PACKER_PLUGIN_AMZ_VERSION)/$(PACKER_PLUGIN_AMZ_ARCHIVE)

SYSTEMD_BOOT_VERSION := 252.12-1~deb12u1
SYSTEMD_BOOT_ARCHIVE := systemd-boot-efi_$(SYSTEMD_BOOT_VERSION)_amd64.deb
SYSTEMD_BOOT_URL := https://snapshot.debian.org/archive/debian/20230712T091300Z/pool/main/s/systemd/$(SYSTEMD_BOOT_ARCHIVE)

UTIL_LINUX_VERSION := 2.39
UTIL_LINUX_SRC := util-linux-$(UTIL_LINUX_VERSION)
UTIL_LINUX_ARCHIVE := $(UTIL_LINUX_SRC).tar.gz
UTIL_LINUX_URL := $(KERNEL_ORG)/utils/util-linux/v$(UTIL_LINUX_VERSION)/$(UTIL_LINUX_ARCHIVE)

CHRONY_VERSION := 4.5
CHRONY_SRC := chrony-$(CHRONY_VERSION)
CHRONY_ARCHIVE := $(CHRONY_SRC).tar.gz
CHRONY_URL := https://chrony-project.org/releases/$(CHRONY_ARCHIVE)
CHRONY_USER := cb-chrony

ZLIB_VERSION := 1.3.1
ZLIB_SRC := zlib-$(ZLIB_VERSION)
ZLIB_ARCHIVE := $(ZLIB_SRC).tar.gz
ZLIB_URL := https://zlib.net/$(ZLIB_ARCHIVE)

OPENSSL_VERSION := 3.2.1
OPENSSL_SRC := openssl-$(OPENSSL_VERSION)
OPENSSL_ARCHIVE := $(OPENSSL_SRC).tar.gz
OPENSSL_URL := https://www.openssl.org/source/$(OPENSSL_ARCHIVE)

OPENSSH_VERSION := V_9_7_P1
OPENSSH_SRC := openssh-portable-$(OPENSSH_VERSION)
OPENSSH_ARCHIVE := $(OPENSSH_VERSION).tar.gz
OPENSSH_URL := https://github.com/openssh/openssh-portable/archive/refs/tags/$(OPENSSH_ARCHIVE)
OPENSSH_PRIVSEP_USER := cb-ssh
OPENSSH_PRIVSEP_DIR := /$(DIR_CB)/empty

HAS_COMMAND_AR := $(DIR_OUT)/.command-ar
HAS_COMMAND_CURL := $(DIR_OUT)/.command-curl
HAS_COMMAND_DOCKER := $(DIR_OUT)/.command-docker
HAS_COMMAND_FAKEROOT := $(DIR_OUT)/.command-fakeroot
HAS_COMMAND_UNZIP := $(DIR_OUT)/.command-unzip
HAS_COMMAND_XZCAT := $(DIR_OUT)/.command-xzcat
HAS_IMAGE_LOCAL := $(DIR_OUT)/.image-local-$(DOCKERFILE_SHA256)

default: release

bootloader: $(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/BOOTX64.EFI

blkid: $(DIR_PREINIT_STG)/$(DIR_CB)/blkid

btrfsprogs: $(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.btrfs

converter: $(DIR_OUT)/converter

e2fsprogs: $(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs $(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext2 \
	$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext3 $(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext4

kernel: $(DIR_KERNEL_STG)/boot/vmlinuz-$(KERNEL_VERSION)

preinit: $(DIR_PREINIT_STG)/$(DIR_CB)/preinit

packer: $(DIR_RELEASE_PACKER)/build.pkr.hcl \
		$(DIR_RELEASE_PACKER)/packer \
		$(DIR_RELEASE_PACKER)/provision \
		$(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE)_SHA256SUM

unpack: $(DIR_OSARCH_BUILD)/unpack

assets-bootloader: $(DIR_RELEASE_ASSETS)/boot.tar

assets-converter: $(DIR_RELEASE_ASSETS)/converter

assets-preinit: $(DIR_RELEASE_ASSETS)/preinit.tar

assets-kernel: $(DIR_RELEASE_ASSETS)/kernel.tar

release-one: $(DIR_RELEASE)/unpack-$(VERSION)-$(OS)-$(ARCH).tar.gz

release:
	for os in linux darwin; do \
		for arch in amd64 arm64; do \
			$(MAKE) $(DIR_OUT)/release/$${os}/$${arch}/unpack-$(VERSION)-$${os}-$${arch}.tar.gz \
				OS=$${os} ARCH=$${arch}; \
		done; \
	done

$(DIR_BOOTLOADER_TMP)/data.tar.xz: $(HAS_COMMAND_AR) $(HAS_COMMAND_XZCAT) \
		$(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE)
	@$(MAKE) $(DIR_BOOTLOADER_TMP)/
	@ar --output $(DIR_BOOTLOADER_TMP) xf $(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE) data.tar.xz

$(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/BOOTX64.EFI: $(DIR_BOOTLOADER_TMP)/data.tar.xz
	@$(MAKE) $(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/
	@xzcat $(DIR_BOOTLOADER_TMP)/data.tar.xz | \
		tar -mxf - \
		--xform "s|.*/systemd-bootx64.efi|$(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/BOOTX64.EFI|" \
		./usr/lib/systemd/boot/efi/systemd-bootx64.efi

$(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE) $(SYSTEMD_BOOT_URL)

$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.btrfs: $(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static
	@$(MAKE) $(DIR_PREINIT_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static $(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.btrfs

$(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(BTRFSPROGS_SRC) \
		hack/compile-btrfsprogs-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(BTRFSPROGS_SRC):/code \
		-e LDFLAGS=-s \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-btrfsprogs-ctr)"
	@touch $(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static

$(DIR_OUT)/$(BTRFSPROGS_SRC): $(HAS_COMMAND_XZCAT) $(DIR_OUT)/$(BTRFSPROGS_ARCHIVE)
	@xzcat $(DIR_OUT)/$(BTRFSPROGS_ARCHIVE) | tar xf - -C $(DIR_OUT)

$(DIR_OUT)/$(BTRFSPROGS_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(BTRFSPROGS_ARCHIVE) $(BTRFSPROGS_URL)

$(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(E2FSPROGS_SRC) \
		hack/compile-e2fsprogs-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(E2FSPROGS_SRC):/code \
		-e LDFLAGS="-s -static" \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-e2fsprogs-ctr)"

$(DIR_OUT)/$(E2FSPROGS_SRC): $(DIR_OUT)/$(E2FSPROGS_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(E2FSPROGS_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(E2FSPROGS_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(E2FSPROGS_ARCHIVE) $(E2FSPROGS_URL)

$(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs: $(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs
	@$(MAKE) $(DIR_PREINIT_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs $(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs

$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext%: $(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs
	@$(MAKE) $(DIR_PREINIT_STG)/$(DIR_CB)/
	@ln -f $(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs $(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext$*

$(DIR_PREINIT_STG)/$(DIR_CB)/preinit: $(HAS_IMAGE_LOCAL) \
		hack/compile-preinit-ctr \
		go.mod \
		$(shell find cmd/preinit -type f -path '*.go' ! -path '*_test.go') \
		$(shell find pkg -type f -path '*.go' ! -path '*_test.go')
	@$(MAKE) $(DIR_PREINIT_STG)/$(DIR_CB)/
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-v $(DIR_ROOT)/$(DIR_PREINIT_STG):/install \
		-e OPENSSH_PRIVSEP_DIR=$(OPENSSH_PRIVSEP_DIR) \
		-e OPENSSH_PRIVSEP_USER=$(OPENSSH_PRIVSEP_USER) \
		-e CHRONY_USER=$(CHRONY_USER) \
		-e DIR_OUT=/install/$(DIR_CB) \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=1 \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-preinit-ctr)"

# Other files are created by the kernel build, but vmlinuz-$(KERNEL_VERSION) will
# be used to indicate the target is created. It is the last file created by the build
# via the $(DIR_ROOT)/kernel/installkernel script mounted in the build container.
$(DIR_KERNEL_STG)/boot/vmlinuz-$(KERNEL_VERSION): $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(KERNEL_SRC) kernel/config \
		hack/compile-kernel-ctr
	@$(MAKE) $(DIR_KERNEL_STG)/boot/ $(DIR_KERNEL_STG)/$(DIR_CB)/
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(KERNEL_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_KERNEL_STG):/install \
		-v $(DIR_ROOT)/kernel/config:/config \
		-v $(DIR_ROOT)/kernel/installkernel:/sbin/installkernel \
		-e INSTALL_PATH=/install/boot \
		-e INSTALL_MOD_PATH=/install/$(DIR_CB) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-kernel-ctr)"
	@rm -f $(DIR_KERNEL_STG)/$(DIR_CB)/lib/modules/$(KERNEL_VERSION)/build
	@rm -f $(DIR_KERNEL_STG)/$(DIR_CB)/lib/modules/$(KERNEL_VERSION)/source

$(DIR_OUT)/$(KERNEL_SRC): $(HAS_COMMAND_XZCAT) $(DIR_OUT)/$(KERNEL_ARCHIVE)
	@xzcat $(DIR_OUT)/$(KERNEL_ARCHIVE) | tar xf - -C $(DIR_OUT)

$(DIR_OUT)/$(KERNEL_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(KERNEL_ARCHIVE) $(KERNEL_URL)

$(DIR_PREINIT_STG)/$(DIR_CB)/amazon.pem: assets/amazon.pem
	@$(MAKE) $(DIR_PREINIT_STG)/$(DIR_CB)/
	@install -m 0644 assets/amazon.pem $(DIR_PREINIT_STG)/$(DIR_CB)/amazon.pem

$(DIR_PREINIT_STG)/$(DIR_CB)/blkid: $(DIR_OUT)/$(UTIL_LINUX_SRC)/blkid.static
	@$(MAKE) $(DIR_PREINIT_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(UTIL_LINUX_SRC)/blkid.static $(DIR_PREINIT_STG)/$(DIR_CB)/blkid

$(DIR_OUT)/$(UTIL_LINUX_SRC)/blkid.static: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(UTIL_LINUX_SRC) \
		hack/compile-blkid-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(UTIL_LINUX_SRC):/code \
		-e CFLAGS=-s \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-blkid-ctr)"

$(DIR_CHRONY_STG)/$(DIR_CB)/chronyd: $(DIR_OUT)/$(CHRONY_SRC)/chronyd
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(CHRONY_SRC)/chronyd $(DIR_CHRONY_STG)/$(DIR_CB)/chronyd

$(DIR_CHRONY_STG)/$(DIR_CB)/chronyc: $(DIR_OUT)/$(CHRONY_SRC)/chronyd
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(CHRONY_SRC)/chronyc $(DIR_CHRONY_STG)/$(DIR_CB)/chronyc

$(DIR_OUT)/$(CHRONY_SRC)/chronyd: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(CHRONY_SRC) \
		hack/compile-chrony-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(CHRONY_SRC):/code \
		-e CHRONY_USER=$(CHRONY_USER) \
		-e SYSCONFDIR=/$(DIR_CB) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-chrony-ctr)"
	@touch $(DIR_OUT)/$(CHRONY_SRC)/chronyd $(DIR_OUT)/$(CHRONY_SRC)/chronyc

$(DIR_CHRONY_STG)/$(DIR_CB)/chrony.conf: assets/chrony.conf
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_CB)/
	@install -m 0644 assets/chrony.conf $(DIR_CHRONY_STG)/$(DIR_CB)/chrony.conf

$(DIR_CHRONY_STG)/$(DIR_CB)/services/chrony:
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_CB)/services/
	@touch $(DIR_CHRONY_STG)/$(DIR_CB)/services/chrony

$(DIR_SSH_STG)/$(DIR_CB)/sftp-server: $(DIR_OUT)/$(OPENSSH_SRC)/sshd
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(OPENSSH_SRC)/sftp-server $(DIR_SSH_STG)/$(DIR_CB)/sftp-server

$(DIR_SSH_STG)/$(DIR_CB)/ssh-keygen: $(DIR_OUT)/$(OPENSSH_SRC)/sshd
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(OPENSSH_SRC)/ssh-keygen $(DIR_SSH_STG)/$(DIR_CB)/ssh-keygen

$(DIR_SSH_STG)/$(DIR_CB)/sshd: $(DIR_OUT)/$(OPENSSH_SRC)/sshd
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(OPENSSH_SRC)/sshd $(DIR_SSH_STG)/$(DIR_CB)/sshd

$(DIR_SSH_STG)/$(DIR_CB)/sshd_config: assets/sshd_config
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_CB)/
	@install -m 0644 assets/sshd_config $(DIR_SSH_STG)/$(DIR_CB)/sshd_config

$(DIR_SSH_STG)/$(DIR_CB)/services/ssh:
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_CB)/services/
	@touch $(DIR_SSH_STG)/$(DIR_CB)/services/ssh

$(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libz.a: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(ZLIB_SRC) \
		hack/compile-zlib-ctr
	@$(MAKE) $(DIR_OUT)/$(DIR_OPENSSH_DEPS)/
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(ZLIB_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(DIR_OPENSSH_DEPS):/$(DIR_OPENSSH_DEPS) \
		-e DIR_OPENSSH_DEPS=/$(DIR_OPENSSH_DEPS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-zlib-ctr)"

$(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libcrypto.a: $(DIR_OUT)/$(OPENSSL_SRC) \
		$(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libz.a hack/compile-openssl-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(OPENSSL_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(DIR_OPENSSH_DEPS):/$(DIR_OPENSSH_DEPS) \
		-e DIR_OPENSSH_DEPS=/$(DIR_OPENSSH_DEPS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-openssl-ctr)"

$(DIR_OUT)/$(OPENSSH_SRC)/sshd: $(DIR_OUT)/$(OPENSSH_SRC) $(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libcrypto.a \
		$(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libz.a hack/compile-openssh-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(OPENSSH_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(DIR_OPENSSH_DEPS):/$(DIR_OPENSSH_DEPS) \
		-e OPENSSH_PRIVSEP_DIR=$(OPENSSH_PRIVSEP_DIR) \
		-e OPENSSH_PRIVSEP_USER=$(OPENSSH_PRIVSEP_USER) \
		-e DIR_OPENSSH_DEPS=/$(DIR_OPENSSH_DEPS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-openssh-ctr)"
	@touch $(DIR_OUT)/$(OPENSSH_SRC)/sshd

# Container image build is done in an empty directory to speed it up.
$(HAS_IMAGE_LOCAL): $(HAS_COMMAND_DOCKER)
	@$(MAKE) $(DIR_OUT)/dockerbuild/
	@docker build \
		--build-arg FROM=$(CTR_IMAGE_GO) \
		--build-arg GID=$$(id -g) \
		--build-arg UID=$$(id -u) \
		-f $(DIR_ROOT)/Dockerfile.build \
		-t $(CTR_IMAGE_LOCAL) \
		$(DIR_OUT)/dockerbuild
	@touch $(HAS_IMAGE_LOCAL)

$(DIR_OUT)/$(UTIL_LINUX_SRC): $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(UTIL_LINUX_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE) $(UTIL_LINUX_URL)

$(DIR_OUT)/$(CHRONY_SRC): $(DIR_OUT)/$(CHRONY_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(CHRONY_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(CHRONY_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(CHRONY_ARCHIVE) $(CHRONY_URL)

$(DIR_OUT)/$(ZLIB_SRC): $(DIR_OUT)/$(ZLIB_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(ZLIB_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(ZLIB_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(ZLIB_ARCHIVE) $(ZLIB_URL)

$(DIR_OUT)/$(OPENSSL_SRC): $(DIR_OUT)/$(OPENSSL_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(OPENSSL_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(OPENSSL_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(OPENSSL_ARCHIVE) $(OPENSSL_URL)

$(DIR_OUT)/$(OPENSSH_SRC): $(DIR_OUT)/$(OPENSSH_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(OPENSSH_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(OPENSSH_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(OPENSSH_ARCHIVE) $(OPENSSH_URL)

$(DIR_RELEASE_ASSETS)/boot.tar: $(HAS_COMMAND_FAKEROOT) $(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/BOOTX64.EFI
	@$(MAKE) $(DIR_RELEASE_ASSETS)/ $(DIR_BOOTLOADER_STG)/boot/loader/entries/
	@chmod -R 0755 $(DIR_BOOTLOADER_STG)
	@cd $(DIR_BOOTLOADER_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/boot.tar boot

$(DIR_RELEASE_ASSETS)/converter: $(DIR_OUT)/converter
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@install -m 0755 $(DIR_OUT)/converter $(DIR_RELEASE_ASSETS)/converter

$(DIR_OUT)/converter: $(HAS_IMAGE_LOCAL) \
		hack/compile-converter-ctr \
		go.mod \
		$(shell find cmd/ctr2ami -type f -path '*.go' ! -path '*_test.go') \
		$(shell find pkg -type f -path '*.go' ! -path '*_test.go')
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-e OPENSSH_PRIVSEP_DIR=$(OPENSSH_PRIVSEP_DIR) \
		-e OPENSSH_PRIVSEP_USER=$(OPENSSH_PRIVSEP_USER) \
		-e CHRONY_USER=$(CHRONY_USER) \
		-e DIR_OUT=/code/$(DIR_OUT) \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-converter-ctr)"

$(DIR_RELEASE_ASSETS)/kernel.tar: $(HAS_COMMAND_FAKEROOT) \
		$(DIR_KERNEL_STG)/boot/vmlinuz-$(KERNEL_VERSION)
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_KERNEL_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/kernel.tar .

$(DIR_RELEASE_ASSETS)/preinit.tar: \
		$(HAS_COMMAND_FAKEROOT) \
		$(DIR_PREINIT_STG)/$(DIR_CB)/amazon.pem \
		$(DIR_PREINIT_STG)/$(DIR_CB)/blkid \
		$(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs \
		$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.btrfs \
		$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext2 \
		$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext3 \
		$(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext4 \
		$(DIR_PREINIT_STG)/$(DIR_CB)/preinit
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_PREINIT_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/preinit.tar .

$(DIR_RELEASE_ASSETS)/chrony.tar: \
		$(HAS_COMMAND_FAKEROOT) \
		$(DIR_CHRONY_STG)/$(DIR_CB)/chrony.conf \
		$(DIR_CHRONY_STG)/$(DIR_CB)/chronyd \
		$(DIR_CHRONY_STG)/$(DIR_CB)/chronyc \
		$(DIR_CHRONY_STG)/$(DIR_CB)/services/chrony
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_CHRONY_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/chrony.tar .

$(DIR_RELEASE_ASSETS)/ssh.tar: \
		$(HAS_COMMAND_FAKEROOT) \
		$(DIR_SSH_STG)/$(DIR_CB)/sftp-server \
		$(DIR_SSH_STG)/$(DIR_CB)/ssh-keygen \
		$(DIR_SSH_STG)/$(DIR_CB)/sshd \
		$(DIR_SSH_STG)/$(DIR_CB)/sshd_config \
		$(DIR_SSH_STG)/$(DIR_CB)/services/ssh
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_SSH_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/ssh.tar .

$(DIR_RELEASE)/unpack-$(VERSION)-$(OS)-$(ARCH).tar.gz: $(HAS_COMMAND_FAKEROOT) packer \
		$(DIR_RELEASE_ASSETS)/boot.tar \
		$(DIR_RELEASE_ASSETS)/converter \
		$(DIR_RELEASE_ASSETS)/preinit.tar \
		$(DIR_RELEASE_ASSETS)/chrony.tar \
		$(DIR_RELEASE_ASSETS)/ssh.tar \
		$(DIR_RELEASE_ASSETS)/kernel.tar \
		$(DIR_RELEASE_BIN)/unpack
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@cd $(DIR_RELEASE) && \
		fakeroot tar czf $(DIR_ROOT)/$(DIR_RELEASE)/unpack-$(VERSION)-$(OS)-$(ARCH).tar.gz assets bin packer

$(DIR_RELEASE_BIN)/unpack: $(DIR_OSARCH_BUILD)/unpack
	@$(MAKE) $(DIR_RELEASE_BIN)/
	@install -m 0755 $(DIR_OSARCH_BUILD)/unpack $(DIR_RELEASE_BIN)/unpack

$(DIR_OSARCH_BUILD)/unpack: $(HAS_IMAGE_LOCAL) \
		hack/compile-unpack-ctr \
		go.mod \
		$(shell find cmd/unpack -type f -path '*.go' ! -path '*_test.go')
	@[ -d $(DIR_OSARCH_BUILD) ] || mkdir -p $(DIR_OSARCH_BUILD)
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-e DIR_OUT=/code/$(DIR_OUT)/osarch/$(OS)/$(ARCH) \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOARCH=$(ARCH) \
		-e GOOS=$(OS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-unpack-ctr)"

$(DIR_RELEASE_PACKER)/packer: $(HAS_COMMAND_UNZIP) $(DIR_OUT)/$(PACKER_ARCHIVE)
	@$(MAKE) $(DIR_RELEASE_PACKER)/
	@unzip -o $(DIR_OUT)/$(PACKER_ARCHIVE) -d $(DIR_RELEASE_PACKER)
	@touch $(DIR_RELEASE_PACKER)/packer

$(DIR_OUT)/$(PACKER_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(PACKER_ARCHIVE) $(PACKER_URL)

$(DIR_RELEASE_PACKER)/build.pkr.hcl: $(DIR_ROOT)/packer/build.pkr.hcl
	@$(MAKE) $(DIR_RELEASE_PACKER)/
	@install -m 0644 $(DIR_ROOT)/packer/build.pkr.hcl $(DIR_RELEASE_PACKER)/build.pkr.hcl

$(DIR_RELEASE_PACKER)/provision: $(DIR_ROOT)/packer/provision
	@install -m 0755 $(DIR_ROOT)/packer/provision $(DIR_RELEASE_PACKER)/provision

$(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE)_SHA256SUM: $(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE)
	@sha256sum $(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE) | \
		awk '{print $1}' > $(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE)_SHA256SUM

$(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE): $(HAS_COMMAND_UNZIP) \
		$(DIR_OUT)/$(PACKER_PLUGIN_AMZ_ARCHIVE)
	@$(MAKE) $(DIR_RELEASE_PACKER_PLUGIN)/
	@unzip -o $(DIR_OUT)/$(PACKER_PLUGIN_AMZ_ARCHIVE) -d $(DIR_RELEASE_PACKER_PLUGIN)
	@touch $(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE)

$(DIR_OUT)/$(PACKER_PLUGIN_AMZ_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -L -o $(DIR_OUT)/$(PACKER_PLUGIN_AMZ_ARCHIVE) $(PACKER_PLUGIN_AMZ_URL)

# Create empty file `$(DIR_OUT)/.command-abc` if command `abc` is found.
$(DIR_OUT)/.command-%:
	@[ -d $(DIR_OUT) ] || mkdir -p $(DIR_OUT)
	@which $* 2>&1 >/dev/null && touch $(DIR_OUT)/.command-$* || (echo "$* is required"; exit 1)

# Create any directory under $(DIR_OUT) as long as it ends in a `/` character.
$(DIR_OUT)/%/:
	@[ -d $(DIR_OUT)/$* ] || mkdir -p $(DIR_OUT)/$*

clean-converter:
	@rm -f $(DIR_OUT)/converter

clean-blkid:
	@rm -f $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE)
	@rm -f $(DIR_PREINIT_STG)/$(DIR_CB)/blkid
	@rm -rf $(DIR_OUT)/$(UTIL_LINUX_SRC)

	@rm -f $(DIR_PREINIT_STG)/$(DIR_CB)/blkid
	@rm -f $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE)
	@rm -rf $(DIR_OUT)/$(UTIL_LINUX_SRC)

clean-e2fsprogs:
	@rm -f $(DIR_OUT)/$(E2FSPROGS_ARCHIVE)
	@rm -f $(DIR_PREINIT_STG)/$(DIR_CB)/mke2fs $(DIR_PREINIT_STG)/$(DIR_CB)/mkfs.ext*
	@rm -rf $(DIR_OUT)/$(E2FSPROGS_SRC)

clean-kernel:
	@rm -f $(DIR_OUT)/$(KERNEL_ARCHIVE)
	@rm -rf $(DIR_OUT)/$(KERNEL_SRC)
	@rm -rf $(DIR_KERNEL_STG)

clean-preinit:
	@rm -f $(DIR_PREINIT_STG)/$(DIR_CB)/preinit

clean:
	@chmod -R +w $(DIR_OUT)/go
	@rm -rf $(DIR_OUT)
