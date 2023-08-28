PROJECT := $(shell basename ${PWD})

DIR_ROOT := $(shell echo ${PWD})
DIR_OUT := $(DIR_ROOT)/_output
DIR_CB := __cb__
DIR_OUT_CB := $(DIR_ROOT)/_output/$(DIR_CB)
DIR_BOOTLOADER := $(DIR_OUT)/bootloader
DIR_PREINIT := $(DIR_OUT)/preinit
DIR_KERNEL := $(DIR_OUT)/kernel
DIR_RELEASE := $(DIR_OUT)/release

COMMIT_ID_HEAD := $(shell git rev-parse HEAD)
CTR_IMAGE_GO := golang:1.21.0-alpine3.18
CTR_IMAGE_LOCAL := $(PROJECT):$(COMMIT_ID_HEAD)

E2FSPROGS_VERSION := 1.47.0
E2FSPROGS_SRC := e2fsprogs-$(E2FSPROGS_VERSION)
E2FSPROGS_ARCHIVE := $(E2FSPROGS_SRC).tar.gz
E2FSPROGS_URL := https://cdn.kernel.org/pub/linux/kernel/people/tytso/e2fsprogs/v$(E2FSPROGS_VERSION)/$(E2FSPROGS_ARCHIVE)

KERNEL_VERSION := 6.1.48
KERNEL_VERSION_MAJ := $(shell echo $(KERNEL_VERSION) | cut -c 1)
KERNEL_SRC := linux-$(KERNEL_VERSION)
KERNEL_ARCHIVE := $(KERNEL_SRC).tar.xz
KERNEL_URL := https://cdn.kernel.org/pub/linux/kernel/v$(KERNEL_VERSION_MAJ).x/$(KERNEL_ARCHIVE)

SYSTEMD_BOOT_VERSION := 252.12-1~deb12u1
SYSTEMD_BOOT_ARCHIVE := systemd-boot-efi_$(SYSTEMD_BOOT_VERSION)_amd64.deb
SYSTEMD_BOOT_URL := https://ftp.debian.org/debian/pool/main/s/systemd/$(SYSTEMD_BOOT_ARCHIVE)

UTIL_LINUX_VERSION := 2.39
UTIL_LINUX_SRC := util-linux-$(UTIL_LINUX_VERSION)
UTIL_LINUX_ARCHIVE := $(UTIL_LINUX_SRC).tar.gz
UTIL_LINUX_URL := https://cdn.kernel.org/pub/linux/utils/util-linux/v$(UTIL_LINUX_VERSION)/$(UTIL_LINUX_ARCHIVE)

HAS_AR := $(DIR_OUT)/.command-ar
HAS_CURL := $(DIR_OUT)/.command-curl
HAS_DOCKER := $(DIR_OUT)/.command-docker
HAS_FAKEROOT := $(DIR_OUT)/.command-fakeroot
HAS_XZCAT := $(DIR_OUT)/.command-xzcat
HAS_IMAGE_LOCAL := $(DIR_OUT)/.image-local-$(COMMIT_ID_HEAD)

default: release

bootloader: $(DIR_BOOTLOADER)/boot/EFI/BOOT/BOOTX64.EFI

blkid: $(DIR_PREINIT)/$(DIR_CB)/blkid

converter: $(HAS_IMAGE_LOCAL)
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-e DIR_OUT=/code/_output \
		-e GOPATH=/code/_output/go \
		-e GOCACHE=/code/_output/gocache \
		-e CGO_ENABLED=0 \
		-w /code/ctr2ami \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-converter-ctr)"

e2fsprogs: $(DIR_PREINIT)/$(DIR_CB)/mke2fs $(DIR_PREINIT)/$(DIR_CB)/mkfs.ext2 \
	$(DIR_PREINIT)/$(DIR_CB)/mkfs.ext3 $(DIR_PREINIT)/$(DIR_CB)/mkfs.ext4

kernel: $(DIR_KERNEL)/boot/vmlinuz-$(KERNEL_VERSION)

preinit: $(HAS_IMAGE_LOCAL)
	@$(MAKE) $(DIR_PREINIT)/$(DIR_CB)/
	@docker run -it \
		-v $(DIR_ROOT):/code \
		-e DIR_OUT=/code/_output/preinit/$(DIR_CB) \
		-e GOPATH=/code/_output/go \
		-e GOCACHE=/code/_output/gocache \
		-e CGO_ENABLED=1 \
		-w /code/preinit \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-preinit-ctr)"

release-bootloader: $(DIR_RELEASE)/boot.tar

release-converter: $(DIR_RELEASE)/converter.tar.gz

release-preinit: $(DIR_RELEASE)/preinit.tar

release-kernel: $(DIR_RELEASE)/kernel-$(KERNEL_VERSION).tar

release: release-bootloader release-converter release-preinit release-kernel

$(DIR_BOOTLOADER)/boot/EFI/BOOT/BOOTX64.EFI: $(HAS_AR) $(HAS_XZCAT) \
		$(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE)
	@$(MAKE) $(DIR_BOOTLOADER)/tmp/ $(DIR_BOOTLOADER)/boot/EFI/BOOT/
	@ar --output $(DIR_BOOTLOADER)/tmp xf $(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE) data.tar.xz
	@xzcat $(DIR_BOOTLOADER)/tmp/data.tar.xz | \
		tar -mxf - \
		--xform "s|.*/systemd-bootx64.efi|_output/bootloader/boot/EFI/BOOT/BOOTX64.EFI|" \
		./usr/lib/systemd/boot/efi/systemd-bootx64.efi

$(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE): $(HAS_CURL)
	@curl -o $(DIR_OUT)/$(SYSTEMD_BOOT_ARCHIVE) $(SYSTEMD_BOOT_URL)

$(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(E2FSPROGS_SRC)
	@docker run -it \
		-v $(DIR_OUT)/$(E2FSPROGS_SRC):/code \
		-e LDFLAGS="-s -static" \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-e2fsprogs-ctr)"

$(DIR_OUT)/$(E2FSPROGS_SRC): $(DIR_OUT)/$(E2FSPROGS_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(E2FSPROGS_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(E2FSPROGS_ARCHIVE): $(HAS_CURL)
	@curl -o $(DIR_OUT)/$(E2FSPROGS_ARCHIVE) $(E2FSPROGS_URL)

$(DIR_PREINIT)/$(DIR_CB)/mke2fs: $(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs
	@$(MAKE) $(DIR_PREINIT)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(E2FSPROGS_SRC)/misc/mke2fs $(DIR_PREINIT)/$(DIR_CB)/mke2fs

$(DIR_PREINIT)/$(DIR_CB)/mkfs.ext%: $(DIR_PREINIT)/$(DIR_CB)/mke2fs
	@$(MAKE) $(DIR_PREINIT)/$(DIR_CB)/
	@ln -f $(DIR_PREINIT)/$(DIR_CB)/mke2fs $(DIR_PREINIT)/$(DIR_CB)/mkfs.ext$*

# Other files are created by the kernel build, but vmlinuz-$(KERNEL_VERSION) will
# be used to indicate the target is created. It is the last file created by the build
# via the $(DIR_ROOT)/kernel/installkernel script mounted in the build container.
$(DIR_KERNEL)/boot/vmlinuz-$(KERNEL_VERSION): $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(KERNEL_SRC)
	@$(MAKE) $(DIR_KERNEL)/boot/ $(DIR_KERNEL)/$(DIR_CB)/
	@docker run -it \
		-v $(DIR_OUT)/$(KERNEL_SRC):/code \
		-v $(DIR_KERNEL):/install \
		-v $(DIR_ROOT)/kernel/config:/config \
		-v $(DIR_ROOT)/kernel/installkernel:/sbin/installkernel \
		-e INSTALL_PATH=/install/boot \
		-e INSTALL_MOD_PATH=/install/$(DIR_CB) \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-kernel-ctr)"
	@rm -f $(DIR_KERNEL)/$(DIR_CB)/lib/modules/$(KERNEL_VERSION)/build
	@rm -f $(DIR_KERNEL)/$(DIR_CB)/lib/modules/$(KERNEL_VERSION)/source

$(DIR_OUT)/$(KERNEL_SRC): $(HAS_XZCAT) $(DIR_OUT)/$(KERNEL_ARCHIVE)
	@xzcat $(DIR_OUT)/$(KERNEL_ARCHIVE) | tar xf - -C $(DIR_OUT)

$(DIR_OUT)/$(KERNEL_ARCHIVE): $(HAS_CURL)
	@curl -o $(DIR_OUT)/$(KERNEL_ARCHIVE) $(KERNEL_URL)

$(DIR_PREINIT)/$(DIR_CB)/blkid: $(DIR_OUT)/$(UTIL_LINUX_SRC)/blkid.static
	@$(MAKE) $(DIR_PREINIT)/$(DIR_CB)/
	@install -m 0755 $(DIR_OUT)/$(UTIL_LINUX_SRC)/blkid.static $(DIR_PREINIT)/$(DIR_CB)/blkid

$(DIR_OUT)/$(UTIL_LINUX_SRC)/blkid.static: $(HAS_IMAGE_LOCAL) $(DIR_OUT)/$(UTIL_LINUX_SRC)
	@docker run -it \
		-v $(DIR_ROOT)/_output/$(UTIL_LINUX_SRC):/code \
		-e CFLAGS=-s \
		-w /code \
		$(CTR_IMAGE_LOCAL) /bin/sh -c "$$(cat $(DIR_ROOT)/hack/compile-blkid-ctr)"

# Container image build is done in an empty directory to speed it up.
$(HAS_IMAGE_LOCAL): $(HAS_DOCKER)
	@$(MAKE) $(DIR_OUT)/dockerbuild/
	@docker build --build-arg FROM=$(CTR_IMAGE_GO) \
		--build-arg GID=$$(id -g) \
		--build-arg UID=$$(id -u) \
		-f $(DIR_ROOT)/Dockerfile.build \
		-t $(CTR_IMAGE_LOCAL) \
		$(DIR_OUT)/dockerbuild
	@touch $(HAS_IMAGE_LOCAL)

$(DIR_OUT)/$(UTIL_LINUX_SRC): $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE)
	@tar zxf $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE) -C $(DIR_OUT)

$(DIR_OUT)/$(UTIL_LINUX_ARCHIVE): $(HAS_CURL)
	@curl -o $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE) $(UTIL_LINUX_URL)

$(DIR_RELEASE)/boot.tar: $(HAS_FAKEROOT) $(DIR_BOOTLOADER)/boot/EFI/BOOT/BOOTX64.EFI
	@$(MAKE) $(DIR_RELEASE)/ $(DIR_BOOTLOADER)/boot/loader/entries/
	@chmod -R 0755 $(DIR_BOOTLOADER)
	@cd $(DIR_BOOTLOADER) && fakeroot tar cf $(DIR_RELEASE)/boot.tar boot

$(DIR_RELEASE)/converter.tar.gz: $(HAS_FAKEROOT) $(DIR_OUT)/converter
	@$(MAKE) $(DIR_RELEASE)/
	@cd $(DIR_OUT) && fakeroot tar zcf $(DIR_RELEASE)/converter.tar.gz converter

$(DIR_RELEASE)/kernel-$(KERNEL_VERSION).tar: $(HAS_FAKEROOT) \
		$(DIR_KERNEL)/boot/vmlinuz-$(KERNEL_VERSION)
	@$(MAKE) $(DIR_RELEASE)/
	@cd $(DIR_KERNEL) && fakeroot tar cf $(DIR_RELEASE)/kernel-$(KERNEL_VERSION).tar .

$(DIR_RELEASE)/preinit.tar: \
		$(HAS_FAKEROOT) \
		$(DIR_PREINIT)/$(DIR_CB)/blkid \
		$(DIR_PREINIT)/$(DIR_CB)/mke2fs \
		$(DIR_PREINIT)/$(DIR_CB)/mkfs.ext2 \
		$(DIR_PREINIT)/$(DIR_CB)/mkfs.ext3 \
		$(DIR_PREINIT)/$(DIR_CB)/mkfs.ext4
	@$(MAKE) $(DIR_RELEASE)/
	@cd $(DIR_PREINIT) && fakeroot tar cf $(DIR_RELEASE)/preinit.tar .

# Create empty file `_output/.command-abc` if command `abc` is found.
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
	@rm -f $(DIR_PREINIT)/$(DIR_CB)/blkid
	@rm -rf $(DIR_OUT)/$(UTIL_LINUX_SRC)

	@rm -f $(DIR_PREINIT)/$(DIR_CB)/blkid
	@rm -f $(DIR_OUT)/$(UTIL_LINUX_ARCHIVE)
	@rm -rf $(DIR_OUT)/$(UTIL_LINUX_SRC)

clean-e2fsprogs:
	@rm -f $(DIR_OUT)/$(E2FSPROGS_ARCHIVE)
	@rm -f $(DIR_PREINIT)/$(DIR_CB)/mke2fs $(DIR_PREINIT)/$(DIR_CB)/mkfs.ext*
	@rm -rf $(DIR_OUT)/$(E2FSPROGS_SRC)

clean-kernel:
	@rm -f $(DIR_OUT)/$(KERNEL_ARCHIVE)
	@rm -rf $(DIR_OUT)/$(KERNEL_SRC)
	@rm -rf $(DIR_OUT)/kernel

clean-preinit:
	@rm -f $(DIR_PREINIT)/$(DIR_CB)/preinit

clean:
	@chmod -R +w $(DIR_OUT)/go
	@rm -rf $(DIR_OUT)
