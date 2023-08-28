OUT_DIR = .
EXECUTABLE = init

default: init

init:
	@CGO_ENABLED=1 go build -o $(OUT_DIR)/$(EXECUTABLE) \
		-ldflags '-linkmode external -extldflags -static -s -w' .

clean:
	rm -f $(OUT_DIR)/$(EXECUTABLE)

.PHONY: init clean
