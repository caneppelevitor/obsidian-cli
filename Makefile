BINARY := obsidian
PREFIX ?= /usr/local/bin

.PHONY: build install uninstall clean test lint

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed $(BINARY) to $(PREFIX)/$(BINARY)"

uninstall:
	rm -f $(PREFIX)/$(BINARY)
	@echo "Removed $(BINARY) from $(PREFIX)/$(BINARY)"

clean:
	rm -f $(BINARY)

test:
	go test ./...

lint:
	go vet ./...
