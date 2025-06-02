BINARY_NAME=kubectl-resource
INSTALL_DIR=/usr/local/bin
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
# this installs the binary as a plugin of kubectl
.PHONY: install

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME)
install: build
	chmod +x $(BINARY_NAME)
	sudo mv $(BINARY_NAME) $(INSTALL_DIR)