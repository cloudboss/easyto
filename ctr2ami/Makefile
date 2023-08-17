include ../Makefile.inc.go

PROJECT := punk
IMPORT_PACKAGE := github.com/cloudboss/cb/punk
EMBED_DIR := files

FIRECRACKER_VERSION := v1.3.0
FIRECRACKER_ARCHIVE := firecracker-$(FIRECRACKER_VERSION)-$(ARCH).tgz
FIRECRACKER_URL := https://github.com/firecracker-microvm/firecracker/releases/download/$(FIRECRACKER_VERSION)/$(FIRECRACKER_ARCHIVE)
FIRECRACKER_EXTRACT_DIR := $(OUT_CACHE_DIR)/release-$(FIRECRACKER_VERSION)-$(ARCH)
FIRECRACKER_EXEC := $(FIRECRACKER_EXTRACT_DIR)/firecracker-$(FIRECRACKER_VERSION)-$(ARCH)
FIRECRACKER_SHA256 := 6239fdd8459405b36fbd2657ddaa02e683c4b0fb137e0617b7207f3330ed2709
JAILER_EXEC := $(FIRECRACKER_EXTRACT_DIR)/jailer-$(FIRECRACKER_VERSION)-$(ARCH)
JAILER_SHA256 := 726c267a8fb0bc702ce9ad31c2a045198d303ed2057b0367218855befb88b6f9

default: $(PROJECT)

setup:
	$(Q)mkdir -p $(OUT_BIN_DIR)
	$(Q)mkdir -p $(OUT_CACHE_DIR)

download:
	$(Q)curl -sL -o $(OUT_CACHE_DIR)/$(FIRECRACKER_ARCHIVE) $(FIRECRACKER_URL)
	$(Q)tar zxf $(OUT_CACHE_DIR)/$(FIRECRACKER_ARCHIVE) -C $(OUT_CACHE_DIR)
	$(Q)cp $(FIRECRACKER_EXEC) $(EMBED_DIR)/firecracker
	$(Q)chmod 0755 $(EMBED_DIR)/firecracker
	$(Q)cp $(JAILER_EXEC) $(EMBED_DIR)/jailer
	$(Q)chmod 0755 $(EMBED_DIR)/jailer

ensure-firecracker: setup
	$(Q)if [ -f $(EMBED_DIR)/firecracker -a -f $(EMBED_DIR)/jailer ]; then \
	    if ! sha256sum $(EMBED_DIR)/firecracker | awk "/^$${FIRECRACKER_SHA256}[[:space:]]/" 1>/dev/null; then \
	        $(MAKE) download; \
	    fi; \
	    if ! sha256sum $(EMBED_DIR)/jailer | awk "/^$${JAILER_SHA256}[[:space:]]/" 1>/dev/null; then \
	        $(MAKE) download; \
	    fi; \
	else \
	    $(MAKE) download; \
	fi

$(PROJECT): ensure-firecracker
	$(Q)go build \
	    -ldflags="-X $(IMPORT_PACKAGE)/$(EMBED_DIR).FirecrackerSHA256Sum=$(FIRECRACKER_SHA256) \
	              -X $(IMPORT_PACKAGE)/$(EMBED_DIR).JailerSHA256Sum=$(JAILER_SHA256)" \
	    -o $(OUT_BIN_DIR)/$(PROJECT) \
	    ./cmd/...

clean:
	$(Q)rm -rf _output
	$(Q)rm -f $(EMBED_DIR)/firecracker

.PHONY: default setup download ensure-firecracker $(PROJECT) clean
