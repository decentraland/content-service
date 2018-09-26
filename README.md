# Content Service

## Requirements

- [Go 1.11](https://golang.org/dl/)
- [Docker Compose](https://docs.docker.com/compose/)

## Setup

1. Set (if it's not already set) your `GOPATH` environment variable, you can do this like: `GOPATH=/home/user/go`

## Flags

- `--local`: Use this flag to use local storage.
- `--local-dir <dir>`: Set the directory you want to store files in, by default `/tmp/`. **IMPORTANT**: Your directory must end with `/`.
- `--s3`: Use this flag to use S3 storage (your environment should be set accordingly).

## Running

To run simply you can simply do `make run`. This will start the HTTP server using port 8000 and using local storage.

If you have Go installed locally you can do `go run main.go <flags>`.

## Endpoints

### POST /mappings

This endpoint recieves a request with `Content-Type:multipart/form-data`. Inside the request we will have: metadata signature, metadata, and the files. **Important**: The metadata part needs to be named `metadata` and the metadata signature part needs to be named `signature`.

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
