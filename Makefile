.PHONY: install uninstall update build clean help build-linux build-windows build-darwin build-all
.DEFAULT_GOAL := help

BINARY_NAME=salter-aws
INSTALL_DIR=$(shell go env GOPATH)/bin
# INSTALL_DIR=/usr/local/bin
SOURCE=main.go

build:
	go build -o $(BINARY_NAME) $(SOURCE)

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux $(SOURCE)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(SOURCE)

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin $(SOURCE)

build-all: build-linux build-windows build-darwin

install: build
	cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@if [ ! -d ~/.bash_completion.d ]; then mkdir -p ~/.bash_completion.d; fi
	cp salter-aws-completion.bash ~/.bash_completion.d/
	@echo "Bash completion installed to ~/.bash_completion.d/salter-aws-completion.bash"
	@echo "Add 'source ~/.bash_completion.d/salter-aws-completion.bash' to your ~/.bashrc if not already"
	@echo "$(BINARY_NAME) installed to $(INSTALL_DIR)"

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -f ~/.bash_completion.d/salter-aws-completion.bash
	@echo "$(BINARY_NAME) uninstalled from $(INSTALL_DIR)"

update: clean
	git pull origin main
	make install

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux $(BINARY_NAME).exe $(BINARY_NAME)-darwin

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build       - Compile for current OS"
	@echo "  build-linux - Compile for Linux (amd64)"
	@echo "  build-windows - Compile for Windows (amd64)"
	@echo "  build-darwin  - Compile for macOS (amd64)"
	@echo "  build-all   - Compile for all platforms"
	@echo ""
	@echo "Management targets:"
	@echo "  install     - Build and install salter-aws to $(INSTALL_DIR)"
	@echo "  uninstall   - Remove salter-aws from $(INSTALL_DIR)"
	@echo "  update      - Pull latest changes and reinstall"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean       - Remove all binaries"
	@echo "  help        - Show this help message"