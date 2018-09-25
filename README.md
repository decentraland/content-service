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
$> curl 'http://localhost:8000/mappings' -F 'f1=@main.go' -F 'f2=@go.mod' -F 'f3=@README.md'
[
  {
    "name": "go.mod",
    "cid": "https://content-service.s3.amazonaws.com/go.mod"
  },
  {
    "name": "README.md",
    "cid": "https://content-service.s3.amazonaws.com/README.md"
  },
  {
    "name": "main.go",
    "cid": "https://content-service.s3.amazonaws.com/main.go"
  }
]
```

To test `GET /contents/{cid}`:

```
$> curl 'http://localhost:8000/contents/main.go'
<a href="https://content-service.s3.amazonaws.com/maing.go">Moved Permanently</a>.
```
