# Content Service

## Requirements

- [Go 1.11](https://golang.org/dl/)
- [Docker Compose](https://docs.docker.com/compose/)

## Flags

- `--local`: Use this flag to use local storage.
- `--local-dir <dir>`: Set the directory you want to store files in, by default `/tmp/`. **IMPORTANT**: Your directory must end with `/`.
- `--s3`: Use this flag to use S3 storage (your environment should be set accordingly).

## Running

First start Redis

```
$ make ops
```

Then you can either build or build and run:

```
$ make build

$ make run
```

This will start an instance of the content service server using port 8000.

## Endpoints

### POST /mappings

This endpoint recieves a request with `Content-Type:multipart/form-data`. Inside the request we will have: metadata signature, metadata, and the files. **Important**: The metadata part needs to be named `metadata` and the metadata signature part needs to be named `signature`.

## Demo

To run the demo script you need to:

1. Have Redis up `$ make ops`
1. Set your AWS environment variables: `AWS_REGION`, `AWS_ACCESS_KEY`, `AWS_SECRET_KEY`
1. Start the demo server `$ make demo`

After, you can run the demo script as:

```
$ ./demo.sh`.
<a href="https://content-service.s3.amazonaws.com/text.txt">Moved Permanently</a>.
```
