# dotenvx decryption for Golang

A tiny Go library to decrypt dotenvx encrypted `.env` files with minimal dependencies.
Works in Docker containers without installing Node.js or dotenvx CLI.

## Package Usage

```go
import "github.com/ericpollmann/dotenvx"

// Compatible with os.Getenv, os.Environ
value := dotenvx.Getenv("MY_SECRET")
envs := dotenvx.Environ()
```

## Minimal working example with Dockerfile

```bash
# Build a tiny working docker image, including go runtime, a 1.33MB image
% make test
go test ./... -cover -count 1
ok  	github.com/ericpollmann/dotenvx	0.256s	coverage: 100.0% of statements
ok  	github.com/ericpollmann/dotenvx/cmd/decrypt	0.150s	coverage: 100.0% of statements
docker build -q -t dotenvx-decrypt .
sha256:e2fd478ccfc868919440bc5442444449a7efb54f293ce1e38c2750419b07ac49
docker images | head -2
REPOSITORY        TAG       IMAGE ID       CREATED          SIZE
dotenvx-decrypt   latest    e2fd478ccfc8   38 seconds ago   1.33MB
docker run -e DOTENV_PRIVATE_KEY="2ff9d3716a37e630e0643447beac508a1e9963444d3ca00a6a22dbf2970dc03d" dotenvx-decrypt
DOTENV_PUBLIC_KEY=020c5f23e6e02f087af380212814755c22f3d742b218666642d1dec184b7c6ae69
GREETING=hello
docker run -e DOTENV_PRIVATE_KEY_PRODUCTION="7d797417f477635f8753c5325d5a68552ab7048f46c518be7f0ae3bc245d3ab8" dotenvx-decrypt
DOTENV_PUBLIC_KEY_PRODUCTION=03f3775e90efd546ad247a3fdcc0d9ef664743579fdd4f7e6c5e6bd73c61f6dc54
GREETING=world
```

## Files

- `decrypt.go` - Decrypts `encrypted:` values using ECIES
- `Dockerfile` - Example multi-stage build with UPX compression (1.33MB binary)
- `go.mod` / `go.sum` - Dependencies (uses `github.com/ecies/go/v2`)

## How It Works

1. Checks for `DOTENV_PRIVATE_KEY_PRODUCTION` → uses `.env.production`, falls back to `DOTENV_PRIVATE_KEY` → uses `.env`
2. Parses env file for `encrypted:` prefixed values  
3. Decrypts using ECIES (compatible with eciesjs/dotenvx)
4. No secrets in RAM or environment after use