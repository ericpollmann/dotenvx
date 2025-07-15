# Build stage
FROM golang:alpine AS builder
RUN apk add --no-cache upx
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w -extldflags -static -a -installsuffix' -o decrypt ./cmd/decrypt
RUN upx --best --lzma decrypt

# Final scratch image
FROM scratch
COPY --from=builder /app/decrypt /decrypt
COPY .env* /
WORKDIR /
ENTRYPOINT ["/decrypt"]