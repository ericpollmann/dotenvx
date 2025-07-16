# dotenvx decryption for Golang

A tiny Go library to decrypt dotenvx encrypted `.env` files with minimal dependencies.
Works in Docker containers without installing Node.js or dotenvx CLI.

## Package Usage

```go
import "github.com/ericpollmann/dotenvx"

// Compatible with os.Getenv, os.Environ or returns a map
value := dotenvx.Getenv("MY_SECRET")
envs := dotenvx.Environ()
envMap := dotenvx.GetenvMap()
```

## Minimal working example with Dockerfile

```bash
# Build a tiny working docker image, including go runtime, a 1.33MB image
% docker build -t dotenvx-decrypt .
% docker images | head -2
REPOSITORY                                                    TAG               IMAGE ID       CREATED                  SIZE
dotenvx-decrypt                                               latest            c01181d8be84   Less than a second ago   1.33MB

# Development uses .env, production uses .env.production
% docker run -e DOTENV_PRIVATE_KEY="2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d" dotenvx-decrypt
DOTENV_PUBLIC_KEY="020c5f23e6e02f087af380212814755c22f3d742b218666642d1dec184b7c6ae69"
GREETING=hello
% docker run -e DOTENV_PRIVATE_KEY_PRODUCTION="7d797417f477635f8753c5325d5a68552ab7048f46c518be7f0ae3bc245d3ab8" dotenvx-decrypt
DOTENV_PUBLIC_KEY_PRODUCTION="03f3775e90efd546ad247a3fdcc0d9ef664743579fdd4f7e6c5e6bd73c61f6dc54"
GREETING=world

% go test ./... -cover
ok  	github.com/ericpollmann/dotenvx	0.122s	coverage: 100.0% of statements
ok  	github.com/ericpollmann/dotenvx/cmd/decrypt	0.200s	coverage: 100.0% of statements
```

## Files

- `decrypt.go` - Decrypts `encrypted:` values using ECIES
- `Dockerfile` - Example multi-stage build with UPX compression (1.33MB binary)
- `go.mod` / `go.sum` - Dependencies (uses `github.com/ecies/go/v2`)

## How It Works

1. Checks for `DOTENV_PRIVATE_KEY_PRODUCTION` → uses `.env.production`, falls back to `DOTENV_PRIVATE_KEY` → uses `.env`
2. Parses env file for `encrypted:` prefixed values  
3. Decrypts using ECIES (compatible with eciesjs/dotenvx)
