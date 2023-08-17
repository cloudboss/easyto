include ../Makefile.inc.go

default: init

setup:
	$(Q)mkdir -p $(OUT_BIN_DIR)
	$(Q)mkdir -p $(OUT_CACHE_DIR)

init: setup
	$(Q)CGO_ENABLED=0 go build -o $(OUT_BIN_DIR)/init -ldflags="-s -w" .

clean:
	$(Q)rm -rf $(OUT_DIR)

.PHONY: agent clean
