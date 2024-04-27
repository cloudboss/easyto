OUT_DIR = .

default: init

init:
	CGO_ENABLED=0 go build -o $(OUT_DIR)/converter -ldflags '-s -w' ./cmd/...

clean:
	rm -f $(OUT_DIR)/converter

.PHONY: setup init clean
