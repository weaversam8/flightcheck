VERSION ?= dev
PREFIX ?= $(HOME)/.local

.PHONY: build test install uninstall dist clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/flightcheck ./cmd/flchk
	ln -sf flightcheck bin/flchk

test:
	go vet ./...
	go test ./...

# Install the binaries into $(PREFIX)/bin and register the Claude Code hooks.
#   make install                  # ~/.local/bin
#   sudo make install PREFIX=/usr/local
install: build
	mkdir -p $(PREFIX)/bin
	install -m 755 bin/flightcheck $(PREFIX)/bin/flightcheck
	ln -sf flightcheck $(PREFIX)/bin/flchk
	$(PREFIX)/bin/flightcheck install

# Remove the hooks and the binaries (keeps ~/.flightcheck).
uninstall:
	-$(PREFIX)/bin/flightcheck uninstall
	rm -f $(PREFIX)/bin/flightcheck $(PREFIX)/bin/flchk

dist:
	scripts/dist.sh $(VERSION)

clean:
	rm -rf bin dist
