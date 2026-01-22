APP_NAME ?= gdrive
CMD_DIR := ./cmd/gdrive
BIN_DIR := bin
GO ?= go
LDFLAGS ?= -s -w
EXEEXT :=

ifeq ($(OS),Windows_NT)
EXEEXT := .exe
endif

.PHONY: build clean tidy test install

build: $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)$(EXEEXT) $(CMD_DIR)

install:
	$(GO) install $(CMD_DIR)

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)

$(BIN_DIR):
	mkdir $(BIN_DIR)
