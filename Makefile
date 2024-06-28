PROJECT := $(shell basename ${PWD})
OS := $(shell uname | tr [:upper:] [:lower:])
ARCH := $(shell arch=$$(uname -m); [ "$${arch}" = "x86_64" ] && echo "amd64" || echo $${arch})
VERSION :=

DIR_ROOT := $(shell echo ${PWD})
DIR_OUT := _output
DIR_ET := .easyto
DIR_BOOTLOADER_TMP := $(DIR_OUT)/bootloader-tmp
DIR_BOOTLOADER_STG := $(DIR_OUT)/staging/bootloader
DIR_CHRONY_STG := $(DIR_OUT)/staging/chrony
DIR_INIT_STG := $(DIR_OUT)/staging/init
DIR_KERNEL_STG := $(DIR_OUT)/staging/kernel
DIR_SSH_STG := $(DIR_OUT)/staging/ssh
DIR_RELEASE := $(DIR_OUT)/release/$(OS)/$(ARCH)
DIR_RELEASE_ASSETS := $(DIR_RELEASE)/assets
DIR_RELEASE_BIN := $(DIR_RELEASE)/bin
DIR_RELEASE_PACKER := $(DIR_RELEASE)/packer
DIR_RELEASE_PACKER_PLUGIN := $(DIR_RELEASE_PACKER)/plugins/github.com/hashicorp/amazon
DIR_OSARCH_BUILD := $(DIR_OUT)/osarch/$(OS)/$(ARCH)
DIR_OPENSSH_DEPS := openssh-deps
DIR_BTRFS_DEPS := btrfs-deps

# $(DIR_BOOTLOADER_STG) not included here as it does not have a $(DIR_ET) subdirectory.
DIRS_STG := $(DIR_CHRONY_STG) $(DIR_INIT_STG) $(DIR_KERNEL_STG) $(DIR_SSH_STG)

DOCKERFILE_SHA256 := $(shell sha256sum Dockerfile.build | awk '{print $$1}' | cut -c 1-40)
CTR_IMAGE_GO := golang:1.22.4-alpine3.20
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

KERNEL_VERSION := 6.6.29
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

BUSYBOX_VERSION := 1.35.0
BUSYBOX_URL := https://www.busybox.net/downloads/binaries/$(BUSYBOX_VERSION)-x86_64-linux-musl/busybox
BUSYBOX_BIN := busybox-$(BUSYBOX_VERSION)

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
OPENSSH_PRIVSEP_DIR := /$(DIR_ET)/var/empty
OPENSSH_DEFAULT_PATH := /$(DIR_ET)/bin:/$(DIR_ET)/sbin:/bin:/usr/bin:/usr/local/bin

SUDO_VERSION := 1.9.15p5
SUDO_SRC := sudo-$(SUDO_VERSION)
SUDO_ARCHIVE := $(SUDO_SRC).tar.gz
SUDO_URL := https://www.sudo.ws/dist/$(SUDO_ARCHIVE)

HAS_COMMAND_AR := $(DIR_OUT)/.command-ar
HAS_COMMAND_CURL := $(DIR_OUT)/.command-curl
HAS_COMMAND_DOCKER := $(DIR_OUT)/.command-docker
HAS_COMMAND_FAKEROOT := $(DIR_OUT)/.command-fakeroot
HAS_COMMAND_UNZIP := $(DIR_OUT)/.command-unzip
HAS_COMMAND_XZCAT := $(DIR_OUT)/.command-xzcat
HAS_IMAGE_LOCAL := $(DIR_OUT)/.image-local-$(DOCKERFILE_SHA256)

VAR_DIR_ET := $(DIR_OUT)/.var-dir-et
VAR_CTR_IMAGE_GO := $(DIR_OUT)/.var-ctr-image-go

default: release

bootloader: $(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/BOOTX64.EFI

blkid: $(DIR_INIT_STG)/$(DIR_ET)/sbin/blkid

btrfsprogs: $(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.btrfs

ctr2disk: $(DIR_OUT)/ctr2disk

e2fsprogs: $(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs $(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext2 \
	$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext3 $(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext4 \
	$(DIR_INIT_STG)/$(DIR_ET)/sbin/resize2fs

kernel: $(DIR_KERNEL_STG)/boot/vmlinuz-$(KERNEL_VERSION)

init: $(DIR_INIT_STG)/$(DIR_ET)/sbin/init

packer: $(DIR_RELEASE_PACKER)/build.pkr.hcl \
		$(DIR_RELEASE_PACKER)/packer \
		$(DIR_RELEASE_PACKER)/provision \
		$(DIR_RELEASE_PACKER_PLUGIN)/$(PACKER_PLUGIN_AMZ_FILE)_SHA256SUM

easyto: $(DIR_OSARCH_BUILD)/easyto

assets-bootloader: $(DIR_RELEASE_ASSETS)/boot.tar

assets-ctr2disk: $(DIR_RELEASE_ASSETS)/ctr2disk

assets-init: $(DIR_RELEASE_ASSETS)/init.tar

assets-kernel: $(DIR_RELEASE_ASSETS)/kernel.tar

release-one: $(DIR_RELEASE)/easyto-$(VERSION)-$(OS)-$(ARCH).tar.gz

release:
	for os in linux darwin; do \
		for arch in amd64 arm64; do \
			$(MAKE) $(DIR_OUT)/release/$${os}/$${arch}/easyto-$(VERSION)-$${os}-$${arch}.tar.gz \
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

$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.btrfs: $(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/sbin/
	@install -m 0755 $(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static $(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.btrfs

$(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(BTRFSPROGS_SRC) \
		$(DIR_OUT)/$(DIR_BTRFS_DEPS)/lib/libblkid.a \
		hack/compile-btrfsprogs-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(BTRFSPROGS_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(DIR_BTRFS_DEPS):/$(DIR_BTRFS_DEPS) \
		-v $(DIR_ROOT)/hack/functions:/functions \
		-e DIR_BTRFS_DEPS=/$(DIR_BTRFS_DEPS) \
		-e PKG_CONFIG_PATH=/$(DIR_BTRFS_DEPS)/lib/pkgconfig \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-btrfsprogs-ctr)"
	@touch $(DIR_OUT)/$(BTRFSPROGS_SRC)/mkfs.btrfs.static

$(DIR_OUT)/$(BTRFSPROGS_SRC): $(HAS_COMMAND_XZCAT) $(DIR_OUT)/$(BTRFSPROGS_ARCHIVE)
	@xzcat $(DIR_OUT)/$(BTRFSPROGS_ARCHIVE) | tar xf - -C $(DIR_OUT)

$(DIR_OUT)/$(BTRFSPROGS_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(BTRFSPROGS_ARCHIVE) $(BTRFSPROGS_URL)

$(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs $(DIR_OUT)/$(E2FSPROGS_SRC)/resize/resize2fs &: $(HAS_IMAGE_LOCAL) \
		$(DIR_OUT)/$(E2FSPROGS_SRC) hack/compile-e2fsprogs-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(E2FSPROGS_SRC):/code \
		-e LDFLAGS="-s -static" \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-e2fsprogs-ctr)"

$(DIR_OUT)/$(E2FSPROGS_SRC): $(DIR_OUT)/$(E2FSPROGS_ARCHIVE)
	@tar zxmf $(DIR_OUT)/$(E2FSPROGS_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(E2FSPROGS_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(E2FSPROGS_ARCHIVE) $(E2FSPROGS_URL)

$(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs: $(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/sbin/
	@install -m 0755 $(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs $(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs

$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext%: $(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/sbin/
	@ln -f $(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs $(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext$*

$(DIR_INIT_STG)/$(DIR_ET)/sbin/resize2fs: $(DIR_OUT)/$(E2FSPROGS_SRC)/resize/resize2fs $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/sbin/
	@install -m 0755 $(DIR_OUT)/$(E2FSPROGS_SRC)/resize/resize2fs $(DIR_INIT_STG)/$(DIR_ET)/sbin/resize2fs

$(DIR_INIT_STG)/$(DIR_ET)/sbin/init: $(HAS_IMAGE_LOCAL) \
		$(VAR_DIR_ET) \
		hack/compile-init-ctr \
		go.mod \
		$(shell find cmd/initial -type f -path '*.go' ! -path '*_test.go') \
		$(shell find pkg -type f -path '*.go' ! -path '*_test.go')
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/sbin/
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-v $(DIR_ROOT)/$(DIR_INIT_STG):/install \
		-e OPENSSH_PRIVSEP_DIR=$(OPENSSH_PRIVSEP_DIR) \
		-e OPENSSH_PRIVSEP_USER=$(OPENSSH_PRIVSEP_USER) \
		-e CHRONY_USER=$(CHRONY_USER) \
		-e DIR_ET_ROOT=/$(DIR_ET) \
		-e DIR_OUT=/install/$(DIR_ET)/sbin \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=1 \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-init-ctr)"

# Other files are created by the kernel build, but vmlinuz-$(KERNEL_VERSION) will
# be used to indicate the target is created. It is the last file created by the build
# via the $(DIR_ROOT)/kernel/installkernel script mounted in the build container.
$(DIR_KERNEL_STG)/boot/vmlinuz-$(KERNEL_VERSION): $(HAS_IMAGE_LOCAL) \
		$(DIR_OUT)/$(KERNEL_SRC) \
		$(VAR_DIR_ET) \
		kernel/config \
		hack/compile-kernel-ctr
	@$(MAKE) $(DIR_KERNEL_STG)/boot/ $(DIR_KERNEL_STG)/$(DIR_ET)/
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(KERNEL_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_KERNEL_STG):/install \
		-v $(DIR_ROOT)/kernel/config:/config \
		-v $(DIR_ROOT)/kernel/installkernel:/sbin/installkernel \
		-e INSTALL_PATH=/install/boot \
		-e INSTALL_MOD_PATH=/install/$(DIR_ET) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-kernel-ctr)"
	@rm -f $(DIR_KERNEL_STG)/$(DIR_ET)/lib/modules/$(KERNEL_VERSION)/build
	@rm -f $(DIR_KERNEL_STG)/$(DIR_ET)/lib/modules/$(KERNEL_VERSION)/source

$(DIR_OUT)/$(KERNEL_SRC): $(HAS_COMMAND_XZCAT) $(DIR_OUT)/$(KERNEL_ARCHIVE)
	@xzcat $(DIR_OUT)/$(KERNEL_ARCHIVE) | tar xf - -C $(DIR_OUT)

$(DIR_OUT)/$(KERNEL_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(KERNEL_ARCHIVE) $(KERNEL_URL)

$(DIR_INIT_STG)/$(DIR_ET)/etc/amazon.pem: assets/amazon.pem $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/etc/
	@install -m 0644 assets/amazon.pem $(DIR_INIT_STG)/$(DIR_ET)/etc/amazon.pem

$(DIR_INIT_STG)/$(DIR_ET)/sbin/blkid: $(DIR_OUT)/$(DIR_BTRFS_DEPS)/sbin/blkid.static $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/sbin/
	@install -m 0755 $(DIR_OUT)/$(DIR_BTRFS_DEPS)/sbin/blkid.static $(DIR_INIT_STG)/$(DIR_ET)/sbin/blkid

$(DIR_OUT)/$(DIR_BTRFS_DEPS)/sbin/blkid.static $(DIR_OUT)/$(DIR_BTRFS_DEPS)/lib/libblkid.a &: \
		$(HAS_IMAGE_LOCAL) \
		$(DIR_OUT)/$(UTIL_LINUX_SRC) \
		hack/compile-blkid-ctr
	@$(MAKE) $(DIR_OUT)/$(DIR_BTRFS_DEPS)/
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(UTIL_LINUX_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(DIR_BTRFS_DEPS):/$(DIR_BTRFS_DEPS) \
		-e DIR_BTRFS_DEPS=/$(DIR_BTRFS_DEPS) \
		-e CFLAGS=-s \
		-e LOCALSTATEDIR=/$(DIR_ET)/var \
		-e RUNSTATEDIR=/$(DIR_ET)/run \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-blkid-ctr)"

$(DIR_CHRONY_STG)/$(DIR_ET)/sbin/chronyd: $(DIR_OUT)/$(CHRONY_SRC)/chronyd $(VAR_DIR_ET)
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_ET)/sbin/
	@install -m 0755 $(DIR_OUT)/$(CHRONY_SRC)/chronyd $(DIR_CHRONY_STG)/$(DIR_ET)/sbin/chronyd

$(DIR_CHRONY_STG)/$(DIR_ET)/bin/chronyc: $(DIR_OUT)/$(CHRONY_SRC)/chronyd $(VAR_DIR_ET)
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_ET)/bin/
	@install -m 0755 $(DIR_OUT)/$(CHRONY_SRC)/chronyc $(DIR_CHRONY_STG)/$(DIR_ET)/bin/chronyc

$(DIR_OUT)/$(CHRONY_SRC)/chronyd: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(CHRONY_SRC) $(VAR_DIR_ET) \
		hack/compile-chrony-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(CHRONY_SRC):/code \
		-e CHRONY_USER=$(CHRONY_USER) \
		-e SYSCONFDIR=/$(DIR_ET)/etc \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-chrony-ctr)"
	@touch $(DIR_OUT)/$(CHRONY_SRC)/chronyd $(DIR_OUT)/$(CHRONY_SRC)/chronyc

$(DIR_CHRONY_STG)/$(DIR_ET)/etc/chrony.conf: assets/chrony.conf $(VAR_DIR_ET)
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_ET)/etc/
	@install -m 0644 assets/chrony.conf $(DIR_CHRONY_STG)/$(DIR_ET)/etc/chrony.conf

$(DIR_CHRONY_STG)/$(DIR_ET)/services/chrony: $(VAR_DIR_ET)
	@$(MAKE) $(DIR_CHRONY_STG)/$(DIR_ET)/services/
	@touch $(DIR_CHRONY_STG)/$(DIR_ET)/services/chrony

$(DIR_SSH_STG)/$(DIR_ET)/libexec/sftp-server: $(DIR_OUT)/$(OPENSSH_SRC)/sshd $(VAR_DIR_ET)
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/libexec/
	@install -m 0755 $(DIR_OUT)/$(OPENSSH_SRC)/sftp-server $(DIR_SSH_STG)/$(DIR_ET)/libexec/sftp-server

$(DIR_SSH_STG)/$(DIR_ET)/bin/busybox: $(DIR_OUT)/$(BUSYBOX_BIN) $(VAR_DIR_ET)
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/bin/
	@install -m 0755 $(DIR_OUT)/$(BUSYBOX_BIN) $(DIR_SSH_STG)/$(DIR_ET)/bin/busybox

$(DIR_SSH_STG)/$(DIR_ET)/bin/sh: assets/sh $(DIR_SSH_STG)/$(DIR_ET)/bin/busybox $(VAR_DIR_ET)
	$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/bin/
	@sed "s|__ROOT_DIR__|${DIR_ET}|g" assets/sh > $(DIR_SSH_STG)/$(DIR_ET)/bin/sh
	@chmod 0755 $(DIR_SSH_STG)/$(DIR_ET)/bin/sh

$(DIR_SSH_STG)/$(DIR_ET)/bin/ssh-keygen: $(DIR_OUT)/$(OPENSSH_SRC)/sshd $(VAR_DIR_ET)
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/bin/
	@install -m 0755 $(DIR_OUT)/$(OPENSSH_SRC)/ssh-keygen $(DIR_SSH_STG)/$(DIR_ET)/bin/ssh-keygen

$(DIR_SSH_STG)/$(DIR_ET)/sbin/sshd: $(DIR_OUT)/$(OPENSSH_SRC)/sshd $(VAR_DIR_ET)
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/sbin/
	@install -m 0755 $(DIR_OUT)/$(OPENSSH_SRC)/sshd $(DIR_SSH_STG)/$(DIR_ET)/sbin/sshd

$(DIR_SSH_STG)/$(DIR_ET)/etc/ssh/sshd_config: assets/sshd_config $(VAR_DIR_ET)
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/etc/ssh/
	@sed "s|__ROOT_DIR__|${DIR_ET}|g" assets/sshd_config > $(DIR_SSH_STG)/$(DIR_ET)/etc/ssh/sshd_config
	@chmod 0600 $(DIR_SSH_STG)/$(DIR_ET)/etc/ssh/sshd_config

$(DIR_SSH_STG)/$(DIR_ET)/services/ssh: $(VAR_DIR_ET)
	@$(MAKE) $(DIR_SSH_STG)/$(DIR_ET)/services/
	@touch $(DIR_SSH_STG)/$(DIR_ET)/services/ssh

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

$(DIR_OUT)/$(OPENSSH_SRC)/sshd: $(DIR_OUT)/$(OPENSSH_SRC) $(VAR_DIR_ET) \
		$(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libcrypto.a \
		$(DIR_OUT)/$(DIR_OPENSSH_DEPS)/lib/libz.a hack/compile-openssh-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(OPENSSH_SRC):/code \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(DIR_OPENSSH_DEPS):/$(DIR_OPENSSH_DEPS) \
		-e OPENSSH_DEFAULT_PATH=$(OPENSSH_DEFAULT_PATH) \
		-e OPENSSH_PRIVSEP_DIR=$(OPENSSH_PRIVSEP_DIR) \
		-e OPENSSH_PRIVSEP_USER=$(OPENSSH_PRIVSEP_USER) \
		-e DIR_OPENSSH_DEPS=/$(DIR_OPENSSH_DEPS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-openssh-ctr)"
	@touch $(DIR_OUT)/$(OPENSSH_SRC)/sshd

$(DIR_OUT)/$(SUDO_SRC)/src/sudo: $(DIR_OUT)/$(SUDO_SRC) $(VAR_DIR_ET) hack/compile-sudo-ctr
	@docker run -it \
		-v $(DIR_ROOT)/$(DIR_OUT)/$(SUDO_SRC):/code \
		-e DIR_ET_ROOT=/$(DIR_ET) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat hack/compile-sudo-ctr)"
	@touch $(DIR_OUT)/$(SUDO_SRC)/src/sudo

$(DIR_SSH_STG)/$(DIR_ET)/bin/sudo: $(DIR_OUT)/$(SUDO_SRC)/src/sudo $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/bin/
	@install -m 4511 $(DIR_OUT)/$(SUDO_SRC)/src/sudo $(DIR_SSH_STG)/$(DIR_ET)/bin/sudo

$(DIR_SSH_STG)/$(DIR_ET)/etc/sudoers: assets/sudoers $(VAR_DIR_ET)
	@$(MAKE) $(DIR_INIT_STG)/$(DIR_ET)/etc/
	@install -m 0440 assets/sudoers $(DIR_SSH_STG)/$(DIR_ET)/etc/sudoers

# Container image build is done in an empty directory to speed it up.
$(HAS_IMAGE_LOCAL): $(HAS_COMMAND_DOCKER) $(VAR_CTR_IMAGE_GO)
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
	@tar zxmf $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(UTIL_LINUX_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE) $(UTIL_LINUX_URL)

$(DIR_OUT)/$(CHRONY_SRC): $(DIR_OUT)/$(CHRONY_ARCHIVE)
	@tar zxmf $(DIR_OUT)/$(CHRONY_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(CHRONY_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(CHRONY_ARCHIVE) $(CHRONY_URL)

$(DIR_OUT)/$(ZLIB_SRC): $(DIR_OUT)/$(ZLIB_ARCHIVE)
	@tar zxmf $(DIR_OUT)/$(ZLIB_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(ZLIB_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(ZLIB_ARCHIVE) $(ZLIB_URL)

$(DIR_OUT)/$(OPENSSL_SRC): $(DIR_OUT)/$(OPENSSL_ARCHIVE)
	@tar zxmf $(DIR_OUT)/$(OPENSSL_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(OPENSSL_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(OPENSSL_ARCHIVE) $(OPENSSL_URL)

$(DIR_OUT)/$(OPENSSH_SRC): $(DIR_OUT)/$(OPENSSH_ARCHIVE)
	@tar zxmf $(DIR_OUT)/$(OPENSSH_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(OPENSSH_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -L -o $(DIR_OUT)/$(OPENSSH_ARCHIVE) $(OPENSSH_URL)

$(DIR_OUT)/$(BUSYBOX_BIN): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(BUSYBOX_BIN) $(BUSYBOX_URL)

$(DIR_OUT)/$(SUDO_SRC): $(DIR_OUT)/$(SUDO_ARCHIVE)
	@tar zxmf $(DIR_OUT)/$(SUDO_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(SUDO_ARCHIVE): $(HAS_COMMAND_CURL)
	@curl -o $(DIR_OUT)/$(SUDO_ARCHIVE) $(SUDO_URL)

$(DIR_RELEASE_ASSETS)/boot.tar: $(HAS_COMMAND_FAKEROOT) $(DIR_BOOTLOADER_STG)/boot/EFI/BOOT/BOOTX64.EFI
	@$(MAKE) $(DIR_RELEASE_ASSETS)/ $(DIR_BOOTLOADER_STG)/boot/loader/entries/
	@chmod -R 0755 $(DIR_BOOTLOADER_STG)
	@cd $(DIR_BOOTLOADER_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/boot.tar boot

$(DIR_RELEASE_ASSETS)/ctr2disk: $(DIR_OUT)/ctr2disk
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@install -m 0755 $(DIR_OUT)/ctr2disk $(DIR_RELEASE_ASSETS)/ctr2disk

$(DIR_OUT)/ctr2disk: $(HAS_IMAGE_LOCAL) \
		$(VAR_DIR_ET) \
		hack/compile-ctr2disk-ctr \
		go.mod \
		$(shell find cmd/ctr2disk -type f -path '*.go' ! -path '*_test.go') \
		$(shell find pkg -type f -path '*.go' ! -path '*_test.go')
	@docker run -it \
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

$(DIR_RELEASE_ASSETS)/kernel.tar: $(HAS_COMMAND_FAKEROOT) \
		$(DIR_KERNEL_STG)/boot/vmlinuz-$(KERNEL_VERSION)
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_KERNEL_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/kernel.tar .

$(DIR_RELEASE_ASSETS)/init.tar: \
		$(HAS_COMMAND_FAKEROOT) \
		$(DIR_INIT_STG)/$(DIR_ET)/etc/amazon.pem \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/blkid \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/init \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.btrfs \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext2 \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext3 \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext4 \
		$(DIR_INIT_STG)/$(DIR_ET)/sbin/resize2fs
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_INIT_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/init.tar .

$(DIR_RELEASE_ASSETS)/chrony.tar: \
		$(HAS_COMMAND_FAKEROOT) \
		$(DIR_CHRONY_STG)/$(DIR_ET)/bin/chronyc \
		$(DIR_CHRONY_STG)/$(DIR_ET)/etc/chrony.conf \
		$(DIR_CHRONY_STG)/$(DIR_ET)/sbin/chronyd \
		$(DIR_CHRONY_STG)/$(DIR_ET)/services/chrony
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_CHRONY_STG) && fakeroot tar cf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/chrony.tar .

$(DIR_RELEASE_ASSETS)/ssh.tar: \
		$(HAS_COMMAND_FAKEROOT) \
		$(DIR_SSH_STG)/$(DIR_ET)/bin/busybox \
		$(DIR_SSH_STG)/$(DIR_ET)/libexec/sftp-server \
		$(DIR_SSH_STG)/$(DIR_ET)/bin/sh \
		$(DIR_SSH_STG)/$(DIR_ET)/bin/ssh-keygen \
		$(DIR_SSH_STG)/$(DIR_ET)/sbin/sshd \
		$(DIR_SSH_STG)/$(DIR_ET)/etc/ssh/sshd_config \
		$(DIR_SSH_STG)/$(DIR_ET)/services/ssh \
		$(DIR_SSH_STG)/$(DIR_ET)/bin/sudo \
		$(DIR_SSH_STG)/$(DIR_ET)/etc/sudoers
	@$(MAKE) $(DIR_RELEASE_ASSETS)/
	@cd $(DIR_SSH_STG) && fakeroot tar cpf $(DIR_ROOT)/$(DIR_RELEASE_ASSETS)/ssh.tar .

$(DIR_RELEASE)/easyto-$(VERSION)-$(OS)-$(ARCH).tar.gz: $(HAS_COMMAND_FAKEROOT) packer \
		$(DIR_RELEASE_ASSETS)/boot.tar \
		$(DIR_RELEASE_ASSETS)/chrony.tar \
		$(DIR_RELEASE_ASSETS)/ctr2disk \
		$(DIR_RELEASE_ASSETS)/init.tar \
		$(DIR_RELEASE_ASSETS)/ssh.tar \
		$(DIR_RELEASE_ASSETS)/kernel.tar \
		$(DIR_RELEASE_BIN)/easyto
	@[ -n "$(VERSION)" ] || (echo "VERSION is required"; exit 1)
	@cd $(DIR_RELEASE) && \
		fakeroot tar czf $(DIR_ROOT)/$(DIR_RELEASE)/easyto-$(VERSION)-$(OS)-$(ARCH).tar.gz assets bin packer

$(DIR_RELEASE_BIN)/easyto: $(DIR_OSARCH_BUILD)/easyto
	@$(MAKE) $(DIR_RELEASE_BIN)/
	@install -m 0755 $(DIR_OSARCH_BUILD)/easyto $(DIR_RELEASE_BIN)/easyto

$(DIR_OSARCH_BUILD)/easyto: $(HAS_IMAGE_LOCAL) \
		$(VAR_DIR_ET) \
		hack/compile-easyto-ctr \
		go.mod \
		$(shell find cmd/easyto -type f -path '*.go' ! -path '*_test.go')
	@[ -d $(DIR_OSARCH_BUILD) ] || mkdir -p $(DIR_OSARCH_BUILD)
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-e DIR_ET_ROOT=/$(DIR_ET) \
		-e DIR_OUT=/code/$(DIR_OUT)/osarch/$(OS)/$(ARCH) \
		-e GOPATH=/code/$(DIR_OUT)/go \
		-e GOCACHE=/code/$(DIR_OUT)/gocache \
		-e CGO_ENABLED=0 \
		-e GOARCH=$(ARCH) \
		-e GOOS=$(OS) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-easyto-ctr)"

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

# Create a file to depend on the contents of $(DIR_ET). Remove the staging
# directory if it changes so the old $(DIR_ET) doesn't end up in release tarballs.
$(VAR_DIR_ET): .FORCE
	@if [ ! -f "$(VAR_DIR_ET)" ]; then \
		echo "$(DIR_ET)" > $(VAR_DIR_ET); \
	else \
		dir_et=$$(cat $(VAR_DIR_ET)); \
		if [ "$(DIR_ET)" != "$${dir_et}" ]; then \
			rm -rf $(DIRS_STG); \
			echo "$(DIR_ET)" > $(VAR_DIR_ET); \
		fi; \
	fi

$(VAR_CTR_IMAGE_GO): .FORCE
	@if [ ! -f "$(VAR_CTR_IMAGE_GO)" ]; then \
		echo "$(CTR_IMAGE_GO)" > $(VAR_CTR_IMAGE_GO); \
	else \
		ctr_image_go=$$(cat $(VAR_CTR_IMAGE_GO)); \
		if [ "$(CTR_IMAGE_GO)" != "$${ctr_image_go}" ]; then \
			echo "$(CTR_IMAGE_GO)" > $(VAR_CTR_IMAGE_GO); \
		fi; \
	fi

.FORCE:

# Create any directory under $(DIR_OUT) as long as it ends in a `/` character.
$(DIR_OUT)/%/:
	@[ -d $(DIR_OUT)/$* ] || mkdir -p $(DIR_OUT)/$*

clean-ctr2disk:
	@rm -f $(DIR_OUT)/ctr2disk

clean-blkid:
	@rm -f $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE)
	@rm -f $(DIR_INIT_STG)/$(DIR_ET)/sbin/blkid
	@rm -rf $(DIR_OUT)/$(UTIL_LINUX_SRC)

clean-e2fsprogs:
	@rm -f $(DIR_OUT)/$(E2FSPROGS_ARCHIVE)
	@rm -f $(DIR_INIT_STG)/$(DIR_ET)/sbin/mke2fs $(DIR_INIT_STG)/$(DIR_ET)/sbin/mkfs.ext*
	@rm -rf $(DIR_OUT)/$(E2FSPROGS_SRC)

clean-kernel:
	@rm -f $(DIR_OUT)/$(KERNEL_ARCHIVE)
	@rm -rf $(DIR_OUT)/$(KERNEL_SRC)
	@rm -rf $(DIR_KERNEL_STG)

clean-init:
	@rm -f $(DIR_INIT_STG)/$(DIR_ET)/sbin/init

clean:
	@chmod -R +w $(DIR_OUT)/go
	@rm -rf $(DIR_OUT)
