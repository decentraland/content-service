# Content Service

## Requirements

- [Go 1.11](https://golang.org/dl/)
- [Docker Compose](https://docs.docker.com/compose/)

## Setup

1. Set (if it's not already set) your `GOPATH` environment variable, you can do this like: `GOPATH=/home/user/go`

## Running

To run simply do `make run`.

This will start an HTTP server using port 8000.

## Testing

You can test manually using `curl`.

To test `POST /mappings`:

```
$> curl 'http://localhost:8000/mappings' -F 'f1=@main.go' -F 'f2=@go.mod' -F 'f3=@README.md
[{"name":"README.md","cid":"b335630551682c19a781afebcf4d07bf978fb1f8ac04c6bf87428ed5106870f5"},{"name":"main.go","cid":"2873f79a86c0d8b3335cd7731b0ecf7dd4301eb19a82ef7a1cba7589b5252261"},{"name":"go.mod","cid":"33ef32bf6c23acb95f5902d7097b7a1d5128ca061167ec0716715b0b9eeaa5f6"}]
```

To test `GET /contents/{cid}`:

```
$> curl 'http://localhost:8000/contents/33ef32bf6c23acb95f5902d7097b7a1d5128ca061167ec0716715b0b9eeaa5f6'
module github.com/decentraland/content-service

require github.com/gorilla/mux v1.6.2
```
