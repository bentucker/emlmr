BIN := $(notdir $(CURDIR))
SOURCE_DIR := $(CURDIR)
GO_MAIN := main.go
BUILD_DIR := $(CURDIR)/build
VERSION := $(shell git describe --always --long --dirty="-snapshot")
ARCHIVE := $(BIN)-$(VERSION)

default: all

clean:
	rm -rf $(BUILD_DIR)
	@rm -rf $(ARCHIVE)
	@rm -f $(ARCHIVE).tar.gz

build: $(GO_MAIN)
	mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BIN)

archive: build
ifeq ("$(wildcard $(ARCHIVE).tar.gz)","")
	@ln -s $(BUILD_DIR) $(ARCHIVE)
	@tar -czf $(ARCHIVE).tar.gz $(ARCHIVE)/*
	@rm $(ARCHIVE)
endif

all: build archive
