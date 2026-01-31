LDFLAGS := -ldflags="-s -w"

.PHONY: build clean release tag

build:
	go build $(LDFLAGS) -o bsky-spy .

clean:
	rm -f bsky-spy bsky-spy-*

release: clean
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bsky-spy-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bsky-spy-darwin-amd64 .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bsky-spy-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bsky-spy-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bsky-spy-windows-amd64.exe .

tag:
	@if [ -z "$(v)" ]; then echo "Usage: make tag v=0.1.0"; exit 1; fi
	git tag v$(v)
	git push origin v$(v)
