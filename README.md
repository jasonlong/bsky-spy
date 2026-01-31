# bsky-spy

Create a Bluesky list from someone else's follows. Use this to discover new people to follow by seeing what's in someone else's feed.

## Install

Download the latest binary from [Releases](https://github.com/jasonlong/bsky-spy/releases), or build from source:

```bash
go install github.com/jasonlong/bsky-spy@latest
```

## Setup

You'll need an app password from Bluesky:

1. Go to **Settings > Privacy and Security > App Passwords**
2. Create a new app password (e.g., name it "bsky-spy")
3. Set environment variables:

```bash
export BSKY_HANDLE=you.bsky.social
export BSKY_APP_KEY=xxxx-xxxx-xxxx-xxxx
```

## Usage

```bash
bsky-spy --name "List Name" <handle>
```

The `--name` flag is required. Lists are public, so you can use any name you want.

Examples:
```bash
bsky-spy --name "Tech Folks" techperson.bsky.social
bsky-spy -n "Design Inspiration" designer.bsky.social
```

## How it works

1. Authenticates with your Bluesky account
2. Fetches all follows from the target user
3. Creates a new curatelist on your profile
4. Adds each followed account to the list

The list appears in your profile under **Lists** and can be used as a custom feed.

## Building from source

```bash
git clone https://github.com/jasonlong/bsky-spy
cd bsky-spy
go build -o bsky-spy .
```

### Cross-compile for all platforms

```bash
# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bsky-spy-darwin-arm64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o bsky-spy-darwin-amd64 .

# Linux
GOOS=linux GOARCH=amd64 go build -o bsky-spy-linux-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o bsky-spy-windows-amd64.exe .
```

## Rate limits

The tool adds small delays between API requests to be respectful. For users following many accounts, expect it to take a minute or two.

## License

MIT
